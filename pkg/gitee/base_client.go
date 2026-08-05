package gitee

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimit represents X-RateLimit-* response headers from the Gitee API.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

type baseClient struct {
	httpClient    *http.Client
	baseURL       string
	accessToken   string
	retryConfig   RetryConfig
	lastRateLimit *RateLimit
	rateLimitMu   sync.RWMutex
}

func (b *baseClient) LastRateLimit() *RateLimit {
	b.rateLimitMu.RLock()
	defer b.rateLimitMu.RUnlock()
	if b.lastRateLimit == nil {
		return nil
	}
	cp := *b.lastRateLimit
	return &cp
}

func (b *baseClient) parseRateLimitHeaders(resp *http.Response) {
	limitStr := resp.Header.Get("X-RateLimit-Limit")
	remainStr := resp.Header.Get("X-RateLimit-Remaining")
	if limitStr == "" && remainStr == "" {
		return
	}
	rl := &RateLimit{}
	if v, err := strconv.Atoi(limitStr); err == nil {
		rl.Limit = v
	}
	if v, err := strconv.Atoi(remainStr); err == nil {
		rl.Remaining = v
	}
	if resetStr := resp.Header.Get("X-RateLimit-Reset"); resetStr != "" {
		if ts, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			rl.Reset = time.Unix(ts, 0)
		}
	}
	b.rateLimitMu.Lock()
	b.lastRateLimit = rl
	b.rateLimitMu.Unlock()
}

func (b *baseClient) newRequest(ctx context.Context, method, path string, queryParams map[string]string, body interface{}) (*http.Request, error) {
	u, err := url.Parse(b.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("gitee: invalid URL: %w", err)
	}

	if len(queryParams) > 0 {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("gitee: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Authorization", "Bearer "+b.accessToken)
	return req, nil
}

func (b *baseClient) executeWithRetry(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("gitee: read body: %w", err)
		}
		req.Body.Close()
	}

	var resp *http.Response

	for attempt := 0; attempt <= b.retryConfig.MaxRetries; attempt++ {
		if err := req.Context().Err(); err != nil {
			return nil, fmt.Errorf("gitee: context: %w", err)
		}

		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		var err error
		resp, err = b.httpClient.Do(req)
		if err != nil {
			if attempt < b.retryConfig.MaxRetries {
				backoff := calculateBackoff(attempt, b.retryConfig)
				slog.Debug("retrying request", "attempt", attempt+1, "error", err, "backoff", backoff)
				select {
				case <-time.After(backoff):
				case <-req.Context().Done():
					return nil, fmt.Errorf("gitee: context: %w", req.Context().Err())
				}
				continue
			}
			return nil, fmt.Errorf("gitee: http: %w", err)
		}

		b.parseRateLimitHeaders(resp)

		slog.Debug("HTTP response", "status", resp.StatusCode, "url", req.URL.String())

		if shouldRetryStatus(resp.StatusCode, b.retryConfig) && attempt < b.retryConfig.MaxRetries {
			backoff := calculateBackoff(attempt, b.retryConfig)
			if resp.StatusCode == http.StatusTooManyRequests {
				if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
					backoff = ra
				}
			}
			slog.Debug("retrying request", "attempt", attempt+1, "status", resp.StatusCode, "backoff", backoff)
			resp.Body.Close()
			select {
			case <-time.After(backoff):
			case <-req.Context().Done():
				return nil, fmt.Errorf("gitee: context: %w", req.Context().Err())
			}
			continue
		}

		break
	}

	return resp, nil
}

func (b *baseClient) do(req *http.Request, v interface{}) error {
	slog.Debug("HTTP request", "method", req.Method, "url", req.URL.String())

	resp, err := b.executeWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return parseErrorResponse(resp)
	}
	if v == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("gitee: decode response: %w", err)
	}
	return nil
}

func (b *baseClient) Do(req *http.Request) (*http.Response, error) {
	baseURL, err := url.Parse(b.baseURL)
	if err != nil {
		return nil, fmt.Errorf("gitee: invalid base URL: %w", err)
	}
	if req.Header.Get("Authorization") == "" && sameOrigin(req.URL, baseURL) {
		req.Header.Set("Authorization", "Bearer "+b.accessToken)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", userAgent())
	}
	slog.Debug("HTTP request", "method", req.Method, "url", req.URL.String())

	return b.executeWithRetry(req)
}

func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}
