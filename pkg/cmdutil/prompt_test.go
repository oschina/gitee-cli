package cmdutil

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func TestConfirmDestructiveActionRequiresYesInNonInteractiveMode(t *testing.T) {
	f := &Factory{IOStreams: &iostreams.IOStreams{
		In:     io.NopCloser(strings.NewReader("yes\n")),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}}

	_, err := ConfirmDestructiveAction(f, "Delete it?")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected a --yes requirement, got %v", err)
	}
}
