package user

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runUserCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewUserCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestUserSearchCmd_plainText(t *testing.T) {
	users := []gitee.User{
		{Login: "alice", Name: "Alice Smith", HTMLURL: "https://gitee.com/alice"},
		{Login: "bob", Name: "Bob Jones", HTMLURL: "https://gitee.com/bob"},
	}
	out, err := runUserCmd([]string{"search", "alice"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(users)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected 'alice' in output, got: %s", out)
	}
	if !strings.Contains(out, "Alice Smith") {
		t.Errorf("expected full name in output, got: %s", out)
	}
}

func TestUserSearchCmd_json(t *testing.T) {
	users := []gitee.User{{Login: "alice", Name: "Alice"}}
	out, err := runUserCmd([]string{"search", "alice", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(users)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"alice"`) && !strings.Contains(out, "alice") {
		t.Errorf("expected alice in JSON output, got: %s", out)
	}
}

func TestUserSearchCmd_empty(t *testing.T) {
	out, err := runUserCmd([]string{"search", "nobody"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]gitee.User{})
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No users found") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestUserSearchCmd_requiresArg(t *testing.T) {
	_, err := runUserCmd([]string{"search"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error when no search query provided")
	}
}

func TestUserSearchCmd_queryPassedToAPI(t *testing.T) {
	var capturedQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		json.NewEncoder(w).Encode([]gitee.User{{Login: "dave"}})
	})
	_, err := runUserCmd([]string{"search", "dave"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if capturedQuery != "dave" {
		t.Errorf("expected query param q=dave, got: %q", capturedQuery)
	}
}
