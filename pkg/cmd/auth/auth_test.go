package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runAuthCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewAuthCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestAuthStatusCmd_loggedIn(t *testing.T) {
	user := gitee.User{Login: "alice", Name: "Alice"}
	out, err := runAuthCmd([]string{"status"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(user)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected username in output, got: %s", out)
	}
	if !strings.Contains(out, "Logged in") && !strings.Contains(out, "已登录") {
		t.Errorf("expected logged-in message, got: %s", out)
		t.Errorf("expected logged in message, got: %s", out)
	}
}

func TestAuthStatusCmd_json(t *testing.T) {
	user := gitee.User{Login: "bob", Name: "Bob"}
	out, err := runAuthCmd([]string{"status", "-j"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(user)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"bob"`) && !strings.Contains(out, "bob") {
		t.Errorf("expected JSON with username, got: %s", out)
	}
}

func TestAuthTokenCmd(t *testing.T) {
	t.Setenv("GITEE_TOKEN", "my-test-token")

	tf := cmdtest.NewTestFactory(http.NotFoundHandler())
	defer tf.Close()

	root := NewAuthCmd(tf.Factory)
	root.SetArgs([]string{"token"})
	err := root.Execute()
	out := tf.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "my-test-token") {
		t.Errorf("expected token in output, got: %q", out)
	}
}
