package iostreams

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestSystem(t *testing.T) {
	s := System()
	if s.In == nil || s.Out == nil || s.ErrOut == nil {
		t.Error("System() should return non-nil streams")
	}
	if s.In != os.Stdin {
		t.Error("expected In to be os.Stdin")
	}
	if s.Out != os.Stdout {
		t.Error("expected Out to be os.Stdout")
	}
	if s.ErrOut != os.Stderr {
		t.Error("expected ErrOut to be os.Stderr")
	}
}

func TestTest(t *testing.T) {
	s, in, out, errOut := Test()
	defer in.Close()
	defer out.Close()
	defer errOut.Close()
	defer os.Remove(in.Name())
	defer os.Remove(out.Name())
	defer os.Remove(errOut.Name())

	if s == nil {
		t.Fatal("Test() returned nil IOStreams")
	}
	if s.In == nil || s.Out == nil || s.ErrOut == nil {
		t.Error("Test() streams should be non-nil")
	}
}

func TestIsTerminal_nonTerminal(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    buf,
		ErrOut: buf,
	}
	if s.IsTerminal() {
		t.Error("bytes.Buffer should not be a terminal")
	}
}

func TestIsStdinTerminal_nonTerminal(t *testing.T) {
	s := &IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	if s.IsStdinTerminal() {
		t.Error("NopCloser should not be a terminal")
	}
}

func TestColorEnabled_noColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	s := &IOStreams{Out: &bytes.Buffer{}}
	if s.ColorEnabled() {
		t.Error("expected color disabled when NO_COLOR is set")
	}
}

func TestColorEnabled_dumbTerm(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	s := &IOStreams{Out: &bytes.Buffer{}}
	if s.ColorEnabled() {
		t.Error("expected color disabled on dumb terminal")
	}
}

func TestColorEnabled_nonTerminalOut(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	s := &IOStreams{Out: &bytes.Buffer{}}
	if s.ColorEnabled() {
		t.Error("expected color disabled when Out is not a terminal")
	}
}
