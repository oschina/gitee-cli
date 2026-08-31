package giteego

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is a gitee-go gateway error response (a non-2xx HTTP status). The
// gateway returns JSON error bodies such as:
//
//	{"message":"You exceeded your current quota, please check your plan and
//	 billing details.","type":"insufficient_quota","param":"","code":"insufficient_quota"}
//
// Fields are parsed leniently and Raw always keeps the original body so
// callers can inspect it.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Type       string `json:"type"`
	Message    string `json:"message"`
	Raw        string `json:"-"`
}

// Error renders the error in the stable "giteego: HTTP <status>: <message>"
// shape. For quota errors a short actionable hint is appended so it surfaces
// through every fmt.Errorf("%w") wrap site without extra handling.
func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Raw
	}
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	out := fmt.Sprintf("giteego: HTTP %d: %s", e.StatusCode, msg)
	if e.IsQuota() {
		out += " (insufficient quota: check the gitee-go plan/billing for this account or enterprise)"
	}
	return out
}

// IsQuota reports whether the error is an exhausted-quota response (HTTP 429
// whose body mentions quota, or code/type insufficient_quota). A bare 429 rate
// limit without a quota body is NOT reported as quota.
func (e *APIError) IsQuota() bool {
	if e == nil {
		return false
	}
	if strings.EqualFold(e.Code, "insufficient_quota") || strings.EqualFold(e.Type, "insufficient_quota") {
		return true
	}
	return e.StatusCode == http.StatusTooManyRequests &&
		strings.Contains(strings.ToLower(e.Raw), "quota")
}

// IsPermission reports whether the error is an auth/permission failure
// (401 Unauthorized or 403 Forbidden).
func (e *APIError) IsPermission() bool {
	if e == nil {
		return false
	}
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// isRetryableStatus reports whether a failed HTTP response is worth retrying.
// 5xx and transport errors are transient; 4xx (bad params, auth, quota) are
// not and retrying them only delays the failure.
func isRetryableStatus(code int) bool {
	return code >= 500
}

// parseAPIError builds an APIError from a non-2xx response body. The body is
// parsed as JSON when possible; the raw body is always preserved.
func parseAPIError(status int, body []byte) *APIError {
	e := &APIError{StatusCode: status}
	if len(body) > 0 {
		e.Raw = strings.TrimSpace(string(body))
		if json.Unmarshal(body, e) != nil {
			// Not JSON — keep the raw body as the message.
			e.Message = ""
		}
	}
	if e.Message == "" && e.Raw != "" {
		e.Message = truncate(e.Raw, 512)
	}
	return e
}

// truncate caps a raw body at n bytes so huge dumps never flood the console.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
