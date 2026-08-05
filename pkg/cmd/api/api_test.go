package api

import (
	"strings"
	"testing"
)

func TestBuildURL(t *testing.T) {
	cases := []struct {
		base     string
		endpoint string
		want     string
	}{
		{"https://gitee.com/api/v5", "/user", "https://gitee.com/api/v5/user"},
		{"https://gitee.com/api/v5/", "/user", "https://gitee.com/api/v5/user"},
		{"https://gitee.com/api/v5", "user", "https://gitee.com/api/v5/user"},
		{"https://custom.com", "/repos/owner/repo", "https://custom.com/repos/owner/repo"},
		{"https://gitee.com/api/v5", "https://other.com/api", "https://other.com/api"},
		{"https://gitee.com/api/v5", "http://other.com/api", "http://other.com/api"},
	}
	for _, tc := range cases {
		got := buildURL(tc.base, tc.endpoint)
		if got != tc.want {
			t.Errorf("buildURL(%q, %q) = %q, want %q", tc.base, tc.endpoint, got, tc.want)
		}
	}
}

func TestBuildBody_fields(t *testing.T) {
	r, err := buildBody([]string{"title=Hello", "body=World"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal("expected non-nil reader")
	}
	buf := new(strings.Builder)
	b := make([]byte, 256)
	n, _ := r.Read(b)
	buf.Write(b[:n])
	body := buf.String()
	if !strings.Contains(body, `"title"`) || !strings.Contains(body, `"Hello"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestBuildBody_rawBody(t *testing.T) {
	raw := `{"foo":"bar"}`
	r, err := buildBody(nil, raw)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 64)
	n, _ := r.Read(b)
	if string(b[:n]) != raw {
		t.Errorf("expected %q, got %q", raw, string(b[:n]))
	}
}

func TestBuildBody_noBody(t *testing.T) {
	r, err := buildBody(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if r != nil {
		t.Error("expected nil reader when no fields and no raw body")
	}
}

func TestBuildBody_invalidField(t *testing.T) {
	_, err := buildBody([]string{"noequalssign"}, "")
	if err == nil {
		t.Error("expected error for invalid field format")
	}
}

func TestBuildBody_rejectsRawBodyAndFields(t *testing.T) {
	_, err := buildBody([]string{"title=Hello"}, `{"body":"raw"}`)
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("expected conflicting body input error, got %v", err)
	}
}
