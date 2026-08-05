package cmdutil

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/charmbracelet/huh"

	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

// IsUserCancelled reports whether err represents a user-initiated cancellation
// (Ctrl+C or Escape) from either the survey or huh prompt libraries.
// Use this to silently exit without printing an error message.
func IsUserCancelled(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return true
	}
	s := err.Error()
	return s == "interrupt" || s == "Interrupt"
}

// FlagError represents a user-facing error caused by invalid command-line flags
// or arguments. Callers can use IsFlagError to detect this type.
type FlagError struct {
	Err error
}

func (e *FlagError) Error() string { return e.Err.Error() }
func (e *FlagError) Unwrap() error { return e.Err }

// FlagErrorf creates a FlagError with a formatted message.
func FlagErrorf(format string, args ...interface{}) error {
	return &FlagError{Err: fmt.Errorf(format, args...)}
}

// IsFlagError reports whether err is or wraps a FlagError.
func IsFlagError(err error) bool {
	var fe *FlagError
	return errors.As(err, &fe)
}

func FriendlyError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "Request timed out. Check your network connection and try again."
		}
		return "Network error. Check your connection and try again."
	}

	if strings.Contains(msg, "not logged in") || strings.Contains(msg, "ErrNotLoggedIn") {
		return "Not logged in. Run: gitee auth login"
	}

	var apiErr *gitee.ErrorResponse
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return "Authentication failed (401). Your token may be invalid or expired.\nRun: gitee auth login"
		case 403:
			return "Permission denied (403). You may not have access to this resource."
		case 404:
			return fmt.Sprintf("Not found (404). Check that the resource exists and you have access.\n  %s", apiErr.Message)
		case 422:
			return fmt.Sprintf("Validation failed (422): %s", apiErr.Message)
		}
		if apiErr.StatusCode >= 500 {
			return "Gitee server error. Please try again later."
		}
	}

	return msg
}
