package giteego

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	defaultBaseURL = "https://go-api.gitee.com/gitee-go"
	defaultRetries = 3
)

// Client talks to the gitee-go gateway. baseURL is the gateway base
// (https://{host}/{path?}/gitee-go); servicePath selects the gitee-go service
// (ServiceIPipe, ServiceSA, ...). The versioned controller path (e.g. /rest/v5)
// is part of each request path, not the service segment.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
	servicePath string
	retries     int
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

func WithBaseURL(base string) Option {
	return func(c *Client) { c.baseURL = base }
}

// WithServicePath selects the gitee-go service segment (default ServiceIPipe).
func WithServicePath(path string) Option {
	return func(c *Client) { c.servicePath = strings.Trim(path, "/") }
}

func WithMaxRetries(n int) Option {
	return func(c *Client) { c.retries = n }
}

func NewClient(accessToken string, opts ...Option) *Client {
	c := &Client{
		httpClient:  &http.Client{Timeout: defaultTimeout},
		baseURL:     defaultBaseURL,
		accessToken: accessToken,
		servicePath: "ipipe",
		retries:     defaultRetries,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// newRequest builds a request against baseURL + servicePath + path.
func (c *Client) newRequest(ctx context.Context, method, path string, query map[string]string, body interface{}) (*http.Request, error) {
	fullPath := path
	if c.servicePath != "" {
		fullPath = "/" + c.servicePath + "/" + strings.TrimLeft(path, "/")
	}
	u := c.baseURL + fullPath
	if len(query) > 0 {
		q := make([]string, 0, len(query))
		for k, v := range query {
			q = append(q, k+"="+v)
		}
		u += "?" + strings.Join(q, "&")
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("giteego: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gitee-cli")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	return req, nil
}

// FetchAbsoluteURL performs an authenticated GET on an absolute URL (no baseURL
// / servicePath prepend) and returns the raw body. Used by `pipeline request`
// to fetch log URLs / arbitrary gitee-go endpoints returned in build output.
func (c *Client) FetchAbsoluteURL(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("giteego: invalid URL: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gitee-cli")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		slog.Debug("giteego HTTP request", "method", req.Method, "url", req.URL.String(), "attempt", attempt+1)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			slog.Debug("giteego HTTP request failed", "error", err, "attempt", attempt+1)
			time.Sleep(100 * time.Duration(attempt+1) * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			lastErr = parseAPIError(resp.StatusCode, body)
			slog.Debug("giteego HTTP response", "status", resp.StatusCode, "url", req.URL.String(), "attempt", attempt+1, "error", lastErr)
			if isRetryableStatus(resp.StatusCode) && attempt < c.retries {
				time.Sleep(100 * time.Duration(attempt+1) * time.Millisecond)
				continue
			}
			return nil, lastErr
		}
		return body, nil
	}
	return nil, lastErr
}

// do executes a request and decodes a ResultVO wrapper, returning its data.
func (c *Client) do(req *http.Request, out interface{}) error {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if err := req.Context().Err(); err != nil {
			return err
		}
		slog.Debug("giteego HTTP request", "method", req.Method, "url", req.URL.String(), "attempt", attempt+1)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			slog.Debug("giteego HTTP request failed", "error", err, "attempt", attempt+1)
			time.Sleep(100 * time.Duration(attempt+1) * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		slog.Debug("giteego HTTP response", "status", resp.StatusCode, "url", req.URL.String(), "attempt", attempt+1)

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			lastErr = parseAPIError(resp.StatusCode, body)
			if isRetryableStatus(resp.StatusCode) && attempt < c.retries {
				time.Sleep(100 * time.Duration(attempt+1) * time.Millisecond)
				continue
			}
			return lastErr
		}

		var wrapped struct {
			Code int `json:"code"`
		}
		dec := json.Unmarshal(body, &wrapped)
		if dec == nil && wrapped.Code != 0 {
			return fmt.Errorf("giteego: %s", extractMsg(body))
		}
		if out != nil {
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("giteego: decode response: %w", err)
			}
		}
		return nil
	}
	return lastErr
}

func extractMsg(body []byte) string {
	var w struct {
		Msg string `json:"msg"`
	}
	if json.Unmarshal(body, &w) == nil && w.Msg != "" {
		return w.Msg
	}
	return strings.TrimSpace(string(body))
}
