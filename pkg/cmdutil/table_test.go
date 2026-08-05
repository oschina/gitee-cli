package cmdutil

import (
	"bytes"
	"testing"
)

func TestWriteTable_alignsTerminalDisplayWidth(t *testing.T) {
	rows := [][]string{
		{"#", "TITLE", "STATE"},
		{"I1", "中文", "open"},
		{"I22", "ASCII", "closed"},
	}

	var out bytes.Buffer
	if err := WriteTable(&out, rows); err != nil {
		t.Fatal(err)
	}

	want := "#    TITLE  STATE\n" +
		"I1   中文   open\n" +
		"I22  ASCII  closed\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected table output:\n%q\nwant:\n%q", got, want)
	}
}

func TestWriteTable_empty(t *testing.T) {
	var out bytes.Buffer
	if err := WriteTable(&out, nil); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output, got %q", out.String())
	}
}
