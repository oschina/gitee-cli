package gitee

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		e.Message = errorEnvelopeMessage(body)
	}
	if e.Message == "" {
		e.Message = strings.TrimSpace(string(body))
	}
	return e
}

// errorEnvelopeMessage extracts a human-readable message from Gitee's error
// envelopes, which key the message under "message", "base", or "messages"
// depending on the failure type.
func errorEnvelopeMessage(body []byte) string {
	var env struct {
		Message  string          `json:"message"`
		Base     json.RawMessage `json:"base"`
		Messages json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return ""
	}
	if env.Message != "" {
		return env.Message
	}
	if m := rawJSONStrings(env.Base); m != "" {
		return m
	}
	return rawJSONStrings(env.Messages)
}

func rawJSONStrings(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return strings.Join(arr, "; ")
	}
	return ""
}

// apiErrorEnvelope detects an error reported with a 2xx status. Gitee
// occasionally returns HTTP 200 with a body such as {"base":["..."]} for
// validation failures (for example, an account that has not completed 2FA).
// Without this check the payload decodes into a zero-value result and the
// command reports success.
func apiErrorEnvelope(body []byte, status int) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}
	msg := errorEnvelopeMessage(trimmed)
	if msg == "" {
		return nil
	}
	return &ErrorResponse{StatusCode: status, Message: msg}
}

type ErrorResponse struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("gitee: HTTP %d: %s", e.StatusCode, e.Message)
}
