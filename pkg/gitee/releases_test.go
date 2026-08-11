package gitee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListReleases(t *testing.T) {
	releases := []Release{{ID: 1, TagName: "v1.0.0"}, {ID: 2, TagName: "v2.0.0"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.ListReleases(context.Background(), "owner", "repo", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(got))
	}
	if got[0].TagName != "v1.0.0" {
		t.Errorf("unexpected tag: %s", got[0].TagName)
	}
}

func TestCreateRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var p CreateReleaseParams
		json.NewDecoder(r.Body).Decode(&p)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Release{ID: 10, TagName: p.TagName, Name: p.Name})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.CreateRelease(context.Background(), "owner", "repo", &CreateReleaseParams{
		TagName: "v3.0.0",
		Name:    "Release 3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TagName != "v3.0.0" {
		t.Errorf("unexpected tag: %s", got.TagName)
	}
}

func TestUpdateRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/repos/owner/repo/releases/5" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var params UpdateReleaseParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			t.Fatal(err)
		}
		json.NewEncoder(w).Encode(Release{ID: 5, TagName: params.TagName, Name: params.Name, Body: params.Body})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.UpdateRelease(context.Background(), "owner", "repo", 5, &UpdateReleaseParams{
		TagName: "v2.0.0",
		Name:    "Version 2",
		Body:    "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 5 || got.TagName != "v2.0.0" {
		t.Fatalf("unexpected release: %#v", got)
	}
}

func TestDeleteRelease(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/releases/5" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	if err := c.DeleteRelease(context.Background(), "owner", "repo", 5); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler not called")
	}
}

func TestGetRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/3" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Release{ID: 3, TagName: "v1.2.0"})
	}))
	defer srv.Close()

	c := NewClient("tok", WithBaseURL(srv.URL))
	got, err := c.GetRelease(context.Background(), "owner", "repo", 3)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 3 {
		t.Errorf("unexpected id: %d", got.ID)
	}
}

func TestGetReleaseByTagNullResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/tags/v1.2.0" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "null")
	}))
	defer srv.Close()

	c := NewClient("token", WithBaseURL(srv.URL))
	got, err := c.GetReleaseByTag(context.Background(), "owner", "repo", "v1.2.0")
	if err == nil {
		t.Fatal("expected a not-found error")
	}
	if got != nil {
		t.Fatalf("expected nil release, got %#v", got)
	}
	var apiErr *ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected HTTP 404 error, got %v", err)
	}
}
