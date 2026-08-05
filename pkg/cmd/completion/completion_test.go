package completion

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCmd_bash(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	cmd := NewCompletionCmd(root)
	outBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"bash"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := outBuf.String()
	if !strings.Contains(out, "bash") && len(out) == 0 {
		t.Error("expected non-empty bash completion output")
	}
}

func TestCompletionCmd_zsh(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	cmd := NewCompletionCmd(root)
	outBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"zsh"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if outBuf.Len() == 0 {
		t.Error("expected non-empty zsh completion output")
	}
}

func TestCompletionCmd_fish(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	cmd := NewCompletionCmd(root)
	outBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"fish"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if outBuf.Len() == 0 {
		t.Error("expected non-empty fish completion output")
	}
}

func TestCompletionCmd_powershell(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	cmd := NewCompletionCmd(root)
	outBuf := &bytes.Buffer{}
	cmd.SetOut(outBuf)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"powershell"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if outBuf.Len() == 0 {
		t.Error("expected non-empty powershell completion output")
	}
}

func TestCompletionCmd_invalidShell(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	cmd := NewCompletionCmd(root)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"unknown-shell"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestCompletionCmd_noArgs(t *testing.T) {
	root := &cobra.Command{Use: "gitee"}
	cmd := NewCompletionCmd(root)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when no shell argument provided")
	}
}
