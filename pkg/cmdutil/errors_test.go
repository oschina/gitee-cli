package cmdutil

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func TestFriendlyError_nil(t *testing.T) {
	if got := FriendlyError(nil); got != "" {
		t.Errorf("expected empty string for nil, got %q", got)
	}
}

func TestFriendlyError_notLoggedIn(t *testing.T) {
	err := errors.New("not logged in: run `gitee auth login` first")
	got := FriendlyError(err)
	if got == "" {
		t.Error("expected non-empty message")
	}
	if !contains(got, "gitee auth login") {
		t.Errorf("expected auth login hint, got: %s", got)
	}
}

func TestFriendlyError_401(t *testing.T) {
	err := &gitee.ErrorResponse{StatusCode: 401, Message: "unauthorized"}
	got := FriendlyError(err)
	if !contains(got, "401") {
		t.Errorf("expected 401 in message, got: %s", got)
	}
	if !contains(got, "auth login") {
		t.Errorf("expected auth login hint, got: %s", got)
	}
}

func TestFriendlyError_403(t *testing.T) {
	err := &gitee.ErrorResponse{StatusCode: 403, Message: "forbidden"}
	got := FriendlyError(err)
	if !contains(got, "403") || !contains(got, "Permission denied") {
		t.Errorf("unexpected 403 message: %s", got)
	}
}

func TestFriendlyError_404(t *testing.T) {
	err := &gitee.ErrorResponse{StatusCode: 404, Message: "not found"}
	got := FriendlyError(err)
	if !contains(got, "404") || !contains(got, "Not found") {
		t.Errorf("unexpected 404 message: %s", got)
	}
}

func TestFriendlyError_422(t *testing.T) {
	err := &gitee.ErrorResponse{StatusCode: 422, Message: "title is required"}
	got := FriendlyError(err)
	if !contains(got, "422") || !contains(got, "Validation failed") {
		t.Errorf("unexpected 422 message: %s", got)
	}
}

func TestFriendlyError_5xx(t *testing.T) {
	err := &gitee.ErrorResponse{StatusCode: 500, Message: "internal server error"}
	got := FriendlyError(err)
	if !contains(got, "server error") {
		t.Errorf("unexpected 5xx message: %s", got)
	}
}

func TestFriendlyError_networkTimeout(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Err: &timeoutError{},
	}
	got := FriendlyError(err)
	if !contains(got, "timed out") && !contains(got, "network") {
		t.Errorf("expected timeout message, got: %s", got)
	}
}

func TestFriendlyError_unknown(t *testing.T) {
	err := errors.New("some unknown error")
	got := FriendlyError(err)
	if got != "some unknown error" {
		t.Errorf("expected passthrough for unknown error, got: %s", got)
	}
}

func TestFriendlyError_wrappedErrorResponse(t *testing.T) {
	inner := &gitee.ErrorResponse{StatusCode: 403, Message: "forbidden"}
	err := fmt.Errorf("something failed: %w", inner)
	got := FriendlyError(err)
	if !contains(got, "403") || !contains(got, "Permission denied") {
		t.Errorf("expected 403 friendly message for wrapped error, got: %s", got)
	}
}

func TestFlagErrorf(t *testing.T) {
	err := FlagErrorf("missing %s flag", "--title")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "missing --title flag" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
	if !IsFlagError(err) {
		t.Error("expected IsFlagError to return true")
	}
}

func TestIsFlagError_false(t *testing.T) {
	if IsFlagError(errors.New("other error")) {
		t.Error("expected IsFlagError to return false for non-flag error")
	}
	if IsFlagError(nil) {
		t.Error("expected IsFlagError to return false for nil")
	}
}

func TestFlagError_unwrap(t *testing.T) {
	inner := errors.New("root cause")
	err := &FlagError{Err: inner}
	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to find inner error via Unwrap")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

var _ net.Error = &net.OpError{Op: "dial", Err: &timeoutError{}, Source: nil}
var _ = time.Second
