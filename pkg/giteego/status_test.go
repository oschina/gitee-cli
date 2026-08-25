package giteego

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// statusBody wraps a service-status data payload in the ResultVO envelope.
func statusBody(t *testing.T, data any) string {
	t.Helper()
	return jsonBody(t, data)
}

func TestCheckServiceStatusPath(t *testing.T) {
	var mu sync.Mutex
	var gotMethod, gotPath string
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPath = r.URL.Path
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, statusBody(t, map[string]any{"gitee_go_status": "active"}))
	})
	vo, err := c.CheckServiceStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	wantPath := "/gitee-go/ipipe/rest/v1/billing/gitee-go-service/status"
	if !strings.Contains(gotPath, wantPath) {
		t.Errorf("path = %q, want contains %q", gotPath, wantPath)
	}
	if !vo.Enabled || vo.Status != "active" {
		t.Errorf("vo = %+v, want enabled with status active", vo)
	}
}

func TestCheckServiceStatusError(t *testing.T) {
	c, _ := testClient("tok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "denied")
	})
	if _, err := c.CheckServiceStatus(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "giteego: HTTP 403") {
		t.Errorf("expected HTTP 403 error, got %v", err)
	}
}

// TestParseServiceStatus covers every parseServiceStatus branch.
func TestParseServiceStatus(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		msg         string
		wantEnabled bool
		wantStatus  string
	}{
		// Null/empty data: enabled iff msg is empty.
		{name: "null data with closed msg", data: "null", msg: "closed", wantEnabled: false},
		{name: "null data empty msg", data: "null", wantEnabled: true},
		{name: "empty data", data: "", wantEnabled: true},
		// Non-object data: lenient default enabled.
		{name: "string data is not an object", data: `"active"`, wantEnabled: true},
		{name: "array data is not an object", data: `["a"]`, wantEnabled: true},
		// Frontend gitee_go_status shape.
		{name: "gitee_go_status active", data: `{"gitee_go_status":"active"}`, wantEnabled: true, wantStatus: "active"},
		{name: "gitee_go_status not_open", data: `{"gitee_go_status":"not_open"}`, wantEnabled: false, wantStatus: "not_open"},
		{name: "gitee_go_status closed", data: `{"gitee_go_status":"closed"}`, wantEnabled: false, wantStatus: "closed"},
		{name: "gitee_go_status non-string", data: `{"gitee_go_status":true}`, wantEnabled: true},
		// status key with known open values.
		{name: "status active", data: `{"status":"active"}`, wantEnabled: true, wantStatus: "active"},
		{name: "status OPENED", data: `{"status":"OPENED"}`, wantEnabled: true, wantStatus: "OPENED"},
		{name: "status OPEN", data: `{"status":"OPEN"}`, wantEnabled: true, wantStatus: "OPEN"},
		{name: "status ENABLED", data: `{"status":"ENABLED"}`, wantEnabled: true, wantStatus: "ENABLED"},
		{name: "status SUCCEED", data: `{"status":"SUCCEED"}`, wantEnabled: true, wantStatus: "SUCCEED"},
		{name: "status FAILED", data: `{"status":"FAILED"}`, wantEnabled: false, wantStatus: "FAILED"},
		{name: "status non-string", data: `{"status":12}`, wantEnabled: true},
		// enabled key shapes: bool, string, number.
		{name: "enabled bool true", data: `{"enabled":true}`, wantEnabled: true},
		{name: "enabled bool false", data: `{"enabled":false}`, wantEnabled: false},
		{name: "enabled string true", data: `{"enabled":"true"}`, wantEnabled: true},
		{name: "enabled string 1", data: `{"enabled":"1"}`, wantEnabled: true},
		{name: "enabled string OPENED", data: `{"enabled":"OPENED"}`, wantEnabled: true},
		{name: "enabled string OPEN", data: `{"enabled":"OPEN"}`, wantEnabled: true},
		{name: "enabled string other", data: `{"enabled":"closed"}`, wantEnabled: false},
		{name: "enabled number zero", data: `{"enabled":0}`, wantEnabled: false},
		{name: "enabled number non-zero", data: `{"enabled":1}`, wantEnabled: true},
		// opened key.
		{name: "opened true", data: `{"opened":true}`, wantEnabled: true},
		{name: "opened false", data: `{"opened":false}`, wantEnabled: false},
		// Unknown shape: lenient default enabled.
		{name: "unknown keys default enabled", data: `{"foo":"bar"}`, wantEnabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vo := parseServiceStatus(json.RawMessage(tt.data), tt.msg)
			if vo.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v (data %q)", vo.Enabled, tt.wantEnabled, tt.data)
			}
			if vo.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", vo.Status, tt.wantStatus)
			}
		})
	}
}
