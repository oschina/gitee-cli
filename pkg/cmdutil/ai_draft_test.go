package cmdutil

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func TestConfirmAIDraftUsesGeneratedDraftInNonInteractiveMode(t *testing.T) {
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(strings.NewReader("")),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	draft := &AIDraft{Title: "Generated title", Body: "Generated body"}
	regenerated := false

	got, err := ConfirmAIDraft(
		ios,
		draft,
		"draft-*.md",
		func() (*AIDraft, error) {
			regenerated = true
			return nil, nil
		},
		"unused", "unused", "unused", "unused", "unused", "unused",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != draft {
		t.Fatalf("expected generated draft to be used, got %+v", got)
	}
	if regenerated {
		t.Fatal("non-interactive confirmation should not regenerate")
	}
}
