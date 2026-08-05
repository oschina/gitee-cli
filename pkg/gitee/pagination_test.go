package gitee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestPaginateAll_multiplePages(t *testing.T) {
	callCount := 0
	results, err := paginateAll(context.Background(), func(ctx context.Context, page, perPage int) ([]string, error) {
		callCount++
		switch page {
		case 1:
			full := make([]string, perPage)
			for i := range full {
				full[i] = fmt.Sprintf("item-%d", i)
			}
			return full, nil
		case 2:
			return []string{"last-1", "last-2"}, nil
		default:
			t.Fatalf("unexpected page %d", page)
			return nil, nil
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
	if len(results) != 102 {
		t.Fatalf("expected 102 results, got %d", len(results))
	}
	if results[0] != "item-0" {
		t.Errorf("expected item-0, got %s", results[0])
	}
	if results[101] != "last-2" {
		t.Errorf("expected last-2, got %s", results[101])
	}
}

func TestPaginateAll_emptyFirstPage(t *testing.T) {
	results, err := paginateAll(context.Background(), func(ctx context.Context, page, perPage int) ([]int, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestPaginateAll_errorPropagation(t *testing.T) {
	wantErr := fmt.Errorf("api failure")
	_, err := paginateAll(context.Background(), func(ctx context.Context, page, perPage int) ([]string, error) {
		return nil, wantErr
	})
	if err != wantErr {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestPaginateAll_contextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := paginateAll(ctx, func(ctx context.Context, page, perPage int) ([]string, error) {
		t.Fatal("fetch should not be called")
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestPaginateAll_maxPageLimit(t *testing.T) {
	callCount := 0
	results, err := paginateAll(context.Background(), func(ctx context.Context, page, perPage int) ([]int, error) {
		callCount++
		full := make([]int, perPage)
		return full, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if callCount != maxPages {
		t.Fatalf("expected %d calls, got %d", maxPages, callCount)
	}
	if len(results) != maxPages*100 {
		t.Fatalf("expected %d results, got %d", maxPages*100, len(results))
	}
}

func TestRateLimitParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5000")
		w.Header().Set("X-RateLimit-Remaining", "4999")
		w.Header().Set("X-RateLimit-Reset", "1700000000")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PullRequest{})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	_, err := c.ListPulls(context.Background(), "owner", "repo", nil)
	if err != nil {
		t.Fatal(err)
	}

	rl := c.LastRateLimit()
	if rl == nil {
		t.Fatal("expected rate limit info, got nil")
	}
	if rl.Limit != 5000 {
		t.Errorf("expected limit 5000, got %d", rl.Limit)
	}
	if rl.Remaining != 4999 {
		t.Errorf("expected remaining 4999, got %d", rl.Remaining)
	}
	expectedReset := time.Unix(1700000000, 0)
	if !rl.Reset.Equal(expectedReset) {
		t.Errorf("expected reset %v, got %v", expectedReset, rl.Reset)
	}
}

func TestRateLimitParsing_noHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PullRequest{})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	_, err := c.ListPulls(context.Background(), "owner", "repo", nil)
	if err != nil {
		t.Fatal(err)
	}

	if c.LastRateLimit() != nil {
		t.Fatal("expected nil rate limit when headers absent")
	}
}

func TestRateLimitParsing_errorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5000")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(60*time.Second).Unix(), 10))
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"message": "rate limit exceeded"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	_, _ = c.ListPulls(context.Background(), "owner", "repo", nil)

	rl := c.LastRateLimit()
	if rl == nil {
		t.Fatal("expected rate limit info even on error response")
	}
	if rl.Remaining != 0 {
		t.Errorf("expected remaining 0, got %d", rl.Remaining)
	}
}

func TestListAllPulls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		switch page {
		case 1:
			prs := make([]PullRequest, 100)
			for i := range prs {
				prs[i] = PullRequest{ID: i + 1}
			}
			json.NewEncoder(w).Encode(prs)
		case 2:
			json.NewEncoder(w).Encode([]PullRequest{{ID: 101}, {ID: 102}})
		default:
			json.NewEncoder(w).Encode([]PullRequest{})
		}
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	prs, err := c.ListAllPulls(context.Background(), "owner", "repo", &ListPullsParams{State: "open"})
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 102 {
		t.Fatalf("expected 102 PRs, got %d", len(prs))
	}
}

func TestListAllIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page <= 1 {
			issues := make([]Issue, 50)
			for i := range issues {
				issues[i] = Issue{ID: i + 1}
			}
			json.NewEncoder(w).Encode(issues)
		} else {
			json.NewEncoder(w).Encode([]Issue{})
		}
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	issues, err := c.ListAllIssues(context.Background(), "owner", "repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 50 {
		t.Fatalf("expected 50 issues, got %d", len(issues))
	}
}

func TestListAllUserRepos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page <= 1 {
			json.NewEncoder(w).Encode([]Repository{{ID: 1, Name: "repo1"}})
		} else {
			json.NewEncoder(w).Encode([]Repository{})
		}
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	repos, err := c.ListAllUserRepos(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
}

func TestListAllReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page <= 1 {
			json.NewEncoder(w).Encode([]Release{{ID: 1, TagName: "v1.0"}})
		} else {
			json.NewEncoder(w).Encode([]Release{})
		}
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	releases, err := c.ListAllReleases(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
}
