package gitee

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig controls automatic retry behavior for transient HTTP errors.
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	RetryOn429   bool
	RetryOn5xx   bool
}

func defaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		RetryOn429:   true,
		RetryOn5xx:   true,
	}
}

func shouldRetryStatus(code int, cfg RetryConfig) bool {
	if code == http.StatusTooManyRequests && cfg.RetryOn429 {
		return true
	}
	if code >= 500 && cfg.RetryOn5xx {
		return true
	}
	return false
}

func calculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	delay := cfg.InitialDelay << attempt
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(delay)/2 + 1))
	return delay + jitter
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
