package sshkey

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/cmdtest"
	"gitee.com/oschina/gitee-cli/pkg/gitee"
)

func runSSHKeyCmd(args []string, handler http.Handler) (string, error) {
	tf := cmdtest.NewTestFactory(handler)
	defer tf.Close()
	root := NewSSHKeyCmd(tf.Factory)
	root.SetOut(tf.OutBuf)
	root.SetErr(tf.ErrOutBuf)
	root.SetArgs(args)
	err := root.Execute()
	return tf.Output(), err
}

func TestSSHKeyListCmd_plainText(t *testing.T) {
	keys := []gitee.SSHKey{
		{ID: 1, Key: "ssh-rsa AAAAB3NzaC1yc2E test@host"},
		{ID: 2, Key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 work@laptop"},
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/user") && !strings.Contains(r.URL.Path, "/keys"):
			json.NewEncoder(w).Encode(gitee.User{Login: "alice"})
		default:
			json.NewEncoder(w).Encode(keys)
		}
	})
	out, err := runSSHKeyCmd([]string{"list"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1") {
		t.Errorf("expected key id 1 in output, got: %s", out)
	}
	if !strings.Contains(out, "test@host") {
		t.Errorf("expected comment in output, got: %s", out)
	}
}

func TestSSHKeyListCmd_json(t *testing.T) {
	keys := []gitee.SSHKey{{ID: 42, Key: "ssh-rsa AAAAB3 ci@bot"}}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/keys") {
			json.NewEncoder(w).Encode(keys)
			return
		}
		json.NewEncoder(w).Encode(gitee.User{Login: "bot"})
	})
	out, err := runSSHKeyCmd([]string{"list", "-j"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("expected key id 42 in JSON output, got: %s", out)
	}
}

func TestSSHKeyListCmd_empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/keys") {
			json.NewEncoder(w).Encode([]gitee.SSHKey{})
			return
		}
		json.NewEncoder(w).Encode(gitee.User{Login: "alice"})
	})
	out, err := runSSHKeyCmd([]string{"list"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No SSH keys found") && !strings.Contains(out, "未找到任何 SSH 密钥") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestSSHKeyAddCmd(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			called = true
			json.NewEncoder(w).Encode(gitee.SSHKey{ID: 99, Key: "ssh-rsa AAAA"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runSSHKeyCmd([]string{"add", "--title", "my-key", "--file", "testdata/fake.pub"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected POST request to add SSH key")
	}
	if !strings.Contains(out, "99") {
		t.Errorf("expected key id 99 in output, got: %s", out)
	}
}

func TestSSHKeyAddCmd_readsKeyFromStdin(t *testing.T) {
	const publicKey = "ssh-ed25519 AAAAC3Nza agent@ci\n"
	tf := cmdtest.NewTestFactory(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var params map[string]string
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			t.Fatal(err)
		}
		if params["key"] != publicKey {
			t.Errorf("expected stdin key %q, got %q", publicKey, params["key"])
		}
		json.NewEncoder(w).Encode(gitee.SSHKey{ID: 100, Key: publicKey})
	}))
	defer tf.Close()
	tf.IOStreams.In = io.NopCloser(strings.NewReader(publicKey))

	root := NewSSHKeyCmd(tf.Factory)
	root.SetArgs([]string{"add", "--title", "agent-key", "--file", "-"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tf.Output(), "100") {
		t.Fatalf("expected created key ID, got %q", tf.Output())
	}
}

func TestSSHKeyAddCmd_missingTitle(t *testing.T) {
	_, err := runSSHKeyCmd([]string{"add", "--file", "key.pub"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error when --title is missing")
	}
}

func TestSSHKeyAddCmd_missingFile(t *testing.T) {
	_, err := runSSHKeyCmd([]string{"add", "--title", "my-key"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error when --file is missing")
	}
}

// ── delete ────────────────────────────────────────────────────────────────────

func TestSSHKeyDeleteCmd(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			called = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	out, err := runSSHKeyCmd([]string{"delete", "5", "--yes"}, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected DELETE request")
	}
	if !strings.Contains(out, "Deleted SSH key 5") && !strings.Contains(out, "已删除 SSH 密钥 5") {
		t.Errorf("expected deletion message, got: %s", out)
	}
}

func TestSSHKeyDeleteCmd_requiresYesInNonInteractiveMode(t *testing.T) {
	called := false
	_, err := runSSHKeyCmd([]string{"delete", "5"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected a --yes requirement, got %v", err)
	}
	if called {
		t.Fatal("delete request should not be sent without --yes")
	}
}

func TestSSHKeyDeleteCmd_invalidID(t *testing.T) {
	_, err := runSSHKeyCmd([]string{"delete", "notanumber"}, http.NotFoundHandler())
	if err == nil {
		t.Error("expected error for invalid key id")
	}
}

func TestKeyComment(t *testing.T) {
	cases := []struct {
		key  string
		want string
	}{
		{"ssh-rsa AAAAB3Nz test@host", "test@host"},
		{"ssh-rsa AAAAB3Nz", "-"},
		{"", "-"},
	}
	for _, c := range cases {
		got := keyComment(c.key)
		if got != c.want {
			t.Errorf("keyComment(%q) = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestKeyPreview(t *testing.T) {
	key := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC test@host"
	preview := keyPreview(key)
	if !strings.HasPrefix(preview, "ssh-rsa ") {
		t.Errorf("expected preview to start with key type, got: %s", preview)
	}
}
