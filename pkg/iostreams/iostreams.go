package iostreams

import (
	"io"
	"os"

	"golang.org/x/term"
)

type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer
}

func System() *IOStreams {
	return &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

func Test() (*IOStreams, *os.File, *os.File, *os.File) {
	in, _ := os.CreateTemp("", "")
	out, _ := os.CreateTemp("", "")
	errOut, _ := os.CreateTemp("", "")
	return &IOStreams{In: in, Out: out, ErrOut: errOut}, in, out, errOut
}

func (s *IOStreams) SetQuiet() {
	s.Out = io.Discard
}

func (s *IOStreams) IsTerminal() bool {
	if f, ok := s.Out.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func (s *IOStreams) IsStdinTerminal() bool {
	if f, ok := s.In.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func (s *IOStreams) ColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return s.IsTerminal()
}
