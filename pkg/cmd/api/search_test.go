package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/internal/build"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func testDoc() *swaggerDoc {
	return &swaggerDoc{
		Paths: map[string]map[string]swaggerOp{
			"/v5/repos/{owner}/{repo}/issues": {
				"get": {
					Tags:    []string{"Issues"},
					Summary: "仓库的所有Issues",
					Parameters: []swaggerParam{
						{Name: "owner", In: "path", Required: true},
						{Name: "repo", In: "path", Required: true},
						{Name: "state", In: "query"},
					},
				},
			},
			"/v5/repos/{owner}/{repo}/pulls": {
				"post": {
					Tags:    []string{"Pull Requests"},
					Summary: "创建Pull Request",
					Parameters: []swaggerParam{
						{Name: "owner", In: "path", Required: true},
						{Name: "repo", In: "path", Required: true},
						{Name: "title", In: "form", Required: true},
					},
				},
			},
			"/v5/user": {
				"get": {
					Tags:    []string{"Users"},
					Summary: "获取授权用户信息",
				},
			},
		},
	}
}

func TestSearchEndpoints_byPath(t *testing.T) {
	matches := searchEndpoints(testDoc(), "issues", 0)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'issues', got %d", len(matches))
	}
	if matches[0].Method != "GET" {
		t.Errorf("expected GET, got %s", matches[0].Method)
	}
	// /v5 prefix should be trimmed
	if matches[0].Path != "/repos/{owner}/{repo}/issues" {
		t.Errorf("expected trimmed path, got %q", matches[0].Path)
	}
}

func TestSearchEndpoints_bySummary(t *testing.T) {
	matches := searchEndpoints(testDoc(), "创建Pull", 0)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for '创建Pull', got %d", len(matches))
	}
	if matches[0].Path != "/repos/{owner}/{repo}/pulls" {
		t.Errorf("expected pulls path, got %q", matches[0].Path)
	}
}

func TestSearchEndpoints_caseInsensitive(t *testing.T) {
	matches := searchEndpoints(testDoc(), "PULLS", 0)
	if len(matches) != 1 {
		t.Fatalf("expected 1 case-insensitive match, got %d", len(matches))
	}
}

func TestSearchEndpoints_limit(t *testing.T) {
	doc := &swaggerDoc{
		Paths: map[string]map[string]swaggerOp{
			"/v5/repos/{owner}/{repo}/issues":   {"get": {Summary: "仓库的所有Issues", Tags: []string{"Issues"}}},
			"/v5/repos/{owner}/{repo}/pulls":    {"get": {Summary: "获取Pull Request列表", Tags: []string{"Pull Requests"}}},
			"/v5/repos/{owner}/{repo}/releases": {"get": {Summary: "获取仓库的所有Releases", Tags: []string{"Releases"}}},
			"/v5/user":                          {"get": {Summary: "获取授权用户信息", Tags: []string{"Users"}}},
		},
	}
	matches := searchEndpoints(doc, "repo", 2)
	if len(matches) > 2 {
		t.Errorf("expected at most 2 matches with limit=2, got %d", len(matches))
	}
}

func TestSearchEndpoints_noMatch(t *testing.T) {
	matches := searchEndpoints(testDoc(), "nonexistent", 0)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestSearchEndpoints_emptyKeyword(t *testing.T) {
	matches := searchEndpoints(testDoc(), "  ", 0)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for empty keyword, got %d", len(matches))
	}
}

func TestPrintEndpoints_format(t *testing.T) {
	var buf bytes.Buffer
	matches := []endpointMatch{
		{
			Method:  "GET",
			Path:    "/repos/{owner}/{repo}/issues",
			Summary: "仓库的所有Issues",
			Tags:    "Issues",
			Params: []swaggerParam{
				{Name: "owner", In: "path", Required: true, Type: "string"},
				{Name: "repo", In: "path", Required: true, Type: "string"},
				{Name: "state", In: "query", Type: "string", Enum: []interface{}{"open", "closed"}},
			},
		},
	}
	printEndpoints(&buf, matches)

	out := buf.String()
	for _, want := range []string{"1 matching", "GET", "/repos/{owner}/{repo}/issues", "仓库的所有Issues", "Issues"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
	// required params marked with * and typed
	if !strings.Contains(out, "* owner (string, path)") {
		t.Errorf("expected required+typed param, got:\n%s", out)
	}
	// enum shown
	if !strings.Contains(out, "one of: open|closed") {
		t.Errorf("expected enum in output, got:\n%s", out)
	}
	// optional param not marked with *
	if strings.Contains(out, "* state") {
		t.Errorf("expected optional param 'state' not marked, got:\n%s", out)
	}
}

func TestPrintEndpoints_noMatches(t *testing.T) {
	var buf bytes.Buffer
	printEndpoints(&buf, nil)
	if !strings.Contains(buf.String(), "No matching endpoints") {
		t.Errorf("expected no-match message, got: %s", buf.String())
	}
}

func TestAPICmd_search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != build.UserAgent() {
			t.Errorf("expected %q user agent, got %q", build.UserAgent(), got)
		}
		enc := json.NewEncoder(w)
		enc.Encode(testDoc())
	}))
	defer srv.Close()

	t.Setenv("GITEE_API_PREFIX", srv.URL)

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	f := &cmdutil.Factory{
		IOStreams: &iostreams.IOStreams{
			In:     io.NopCloser(bytes.NewReader(nil)),
			Out:    outBuf,
			ErrOut: errBuf,
		},
	}

	cmd := NewAPICmd(f)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--search", "issues"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(outBuf.String(), "/repos/{owner}/{repo}/issues") {
		t.Errorf("expected endpoint in search output, got:\n%s", outBuf.String())
	}
}

func TestPaginate(t *testing.T) {
	all := []endpointMatch{
		{Path: "/a"}, {Path: "/b"}, {Path: "/c"}, {Path: "/d"}, {Path: "/e"},
	}

	// page 1, limit 2
	page, total := paginate(all, 1, 2)
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(page) != 2 || page[0].Path != "/a" || page[1].Path != "/b" {
		t.Errorf("expected page1 [/a /b], got %v", page)
	}

	// page 3, limit 2 (last page with 1 item)
	page, _ = paginate(all, 3, 2)
	if len(page) != 1 || page[0].Path != "/e" {
		t.Errorf("expected page3 [/e], got %v", page)
	}

	// page beyond total returns empty
	page, _ = paginate(all, 10, 2)
	if len(page) != 0 {
		t.Errorf("expected empty page for page 10, got %v", page)
	}

	// default limit 20
	page, _ = paginate(all, 0, 0)
	if len(page) != 5 {
		t.Errorf("expected all 5 with default limit, got %d", len(page))
	}
}

func TestPrintPagination_singlePage(t *testing.T) {
	var buf bytes.Buffer
	printPagination(&buf, 5, 1, 20)
	if buf.Len() != 0 {
		t.Errorf("expected no pagination output for single page, got: %s", buf.String())
	}
}

func TestPrintPagination_multiPage(t *testing.T) {
	var buf bytes.Buffer
	printPagination(&buf, 28, 2, 10)
	out := buf.String()
	if !strings.Contains(out, "Page 2/3") {
		t.Errorf("expected 'Page 2/3' in output, got: %s", out)
	}
	if !strings.Contains(out, "28 endpoints") {
		t.Errorf("expected total count in output, got: %s", out)
	}
}
