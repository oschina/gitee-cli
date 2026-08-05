package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListPulls(t *testing.T) {
	prs := []PullRequest{
		{Number: 1, Title: "Fix bug", State: "open"},
		{Number: 2, Title: "Add feature", State: "merged"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(prs)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ListPulls(context.Background(), "owner", "repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(got))
	}
	if got[0].Number != 1 || got[1].Title != "Add feature" {
		t.Errorf("unexpected PR data: %+v", got)
	}
}

func TestGetPull(t *testing.T) {
	pr := PullRequest{Number: 42, Title: "My PR", State: "open"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(pr)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetPull(context.Background(), "owner", "repo", 42)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 42 || got.Title != "My PR" {
		t.Errorf("unexpected PR: %+v", got)
	}
}

func TestUpdatePull_close(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var params UpdatePullParams
		json.NewDecoder(r.Body).Decode(&params)
		if params.State != "closed" {
			t.Errorf("expected state=closed, got %s", params.State)
		}
		json.NewEncoder(w).Encode(PullRequest{Number: 1, State: "closed"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.UpdatePull(context.Background(), "owner", "repo", 1, &UpdatePullParams{State: "closed"})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "closed" {
		t.Errorf("expected closed, got %s", got.State)
	}
}

func TestListPulls_withParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		want := map[string]string{
			"state":            "merged",
			"head":             "alice:feature",
			"base":             "main",
			"sort":             "updated",
			"since":            "2026-05-01T00:00:00Z",
			"direction":        "asc",
			"milestone_number": "9",
			"labels":           "bug,performance",
			"page":             "2",
			"per_page":         "5",
			"author":           "alice",
			"assignee":         "bob",
			"tester":           "carol",
		}
		for key, value := range want {
			if got := q.Get(key); got != value {
				t.Errorf("expected %s=%s, got %s", key, value, got)
			}
		}
		json.NewEncoder(w).Encode([]PullRequest{})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	_, err := c.ListPulls(context.Background(), "owner", "repo", &ListPullsParams{
		State:           "merged",
		Head:            "alice:feature",
		Base:            "main",
		Sort:            "updated",
		Since:           "2026-05-01T00:00:00Z",
		Direction:       "asc",
		MilestoneNumber: 9,
		Labels:          "bug,performance",
		Page:            2,
		PerPage:         5,
		Author:          "alice",
		Assignee:        "bob",
		Tester:          "carol",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReviewPull(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/pulls/7/review" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	if err := c.ReviewPull(context.Background(), "owner", "repo", 7, &ReviewPullParams{}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestMergePull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var p MergePullParams
		json.NewDecoder(r.Body).Decode(&p)
		if p.MergeMethod != "squash" {
			t.Errorf("expected merge_method=squash, got %s", p.MergeMethod)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	if err := c.MergePull(context.Background(), "owner", "repo", 1, "squash", false); err != nil {
		t.Fatal(err)
	}
}

func TestMergePull_defaultMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p MergePullParams
		json.NewDecoder(r.Body).Decode(&p)
		if p.MergeMethod != "merge" {
			t.Errorf("expected default merge_method=merge, got %s", p.MergeMethod)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	if err := c.MergePull(context.Background(), "owner", "repo", 1, "", false); err != nil {
		t.Fatal(err)
	}
}
