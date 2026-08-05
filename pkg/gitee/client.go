package gitee

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitee.com/oschina/gitee-cli/internal/build"
)

const (
	defaultBaseURL = "https://gitee.com/api/v5"
	defaultTimeout = 30 * time.Second
)

func userAgent() string {
	return build.UserAgent()
}

type Client struct {
	baseClient
}

type ClientOption func(*Client)

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

func WithBaseURL(base string) ClientOption {
	return func(c *Client) { c.baseURL = base }
}

func WithRetryConfig(cfg RetryConfig) ClientOption {
	return func(c *Client) { c.retryConfig = cfg }
}

func WithMaxRetries(n int) ClientOption {
	return func(c *Client) { c.retryConfig.MaxRetries = n }
}

func NewClient(accessToken string, opts ...ClientOption) *Client {
	c := &Client{
		baseClient: baseClient{
			httpClient: &http.Client{
				Timeout: defaultTimeout,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) > 0 && !sameOrigin(req.URL, via[0].URL) {
						req.Header.Del("Authorization")
					}
					return nil
				},
			},
			baseURL:     defaultBaseURL,
			accessToken: accessToken,
			retryConfig: defaultRetryConfig(),
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func parseErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	e := &ErrorResponse{StatusCode: resp.StatusCode}
	_ = json.Unmarshal(body, e)
	if e.Message == "" {
		e.Message = string(body)
	}
	return e
}

type ErrorResponse struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("gitee: HTTP %d: %s", e.StatusCode, e.Message)
}
