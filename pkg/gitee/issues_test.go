package gitee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRepoIssues(t *testing.T) {
	issues := []Issue{
		{Number: "1", Title: "Bug", State: "open"},
		{Number: "2", Title: "Feature", State: "progressing"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(issues)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ListRepoIssues(context.Background(), "owner", "repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(got))
	}
}

func TestGetIssue(t *testing.T) {
	iss := Issue{Number: "IJEE1", Title: "Hello"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues/IJEE1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(iss)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetIssue(context.Background(), "owner", "repo", "IJEE1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != "IJEE1" {
		t.Errorf("unexpected issue number: %s", got.Number)
	}
}

func TestCreateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var params CreateIssueParams
		json.NewDecoder(r.Body).Decode(&params)
		if params.Repo != "repo" {
			t.Errorf("expected repo field to be set, got %q", params.Repo)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Issue{Number: "1", Title: params.Title})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.CreateIssue(context.Background(), "owner", "repo", &CreateIssueParams{Title: "New Issue"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != "1" {
		t.Errorf("unexpected issue: %+v", got)
	}
}

func TestUpdateIssue_correctEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		wantPath := "/repos/owner/issues/IJEE5"
		if r.URL.Path != wantPath {
			t.Errorf("expected path %s, got %s", wantPath, r.URL.Path)
		}
		var params UpdateIssueParams
		json.NewDecoder(r.Body).Decode(&params)
		if params.Repo != "myrepo" {
			t.Errorf("expected repo=myrepo in body, got %q", params.Repo)
		}
		json.NewEncoder(w).Encode(Issue{Number: "IJEE5", State: params.State})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.UpdateIssue(context.Background(), "owner", "myrepo", "IJEE5", &UpdateIssueParams{State: "closed"})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "closed" {
		t.Errorf("expected closed, got %s", got.State)
	}
}

func TestListRepoIssues_withFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != "closed" {
			t.Errorf("expected state=closed")
		}
		if q.Get("assignee") != "bob" {
			t.Errorf("expected assignee=bob")
		}
		json.NewEncoder(w).Encode([]Issue{})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	_, err := c.ListRepoIssues(context.Background(), "owner", "repo", &ListIssuesParams{State: "closed", Assignee: "bob"})
	if err != nil {
		t.Fatal(err)
	}
}
