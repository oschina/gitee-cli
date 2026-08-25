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

func TestWriteTableBordered(t *testing.T) {
	rows := [][]string{
		{"ID", "BUILD", "FILE"},
		{"8", "6", "ci.yml"},
		{"5", "5", "deploy.yml"},
	}

	var out bytes.Buffer
	if err := WriteTableBordered(&out, rows); err != nil {
		t.Fatal(err)
	}

	want := "│ ID │ BUILD │ FILE       │\n" +
		"├────┼───────┼────────────┤\n" +
		"│ 8  │ 6     │ ci.yml     │\n" +
		"│ 5  │ 5     │ deploy.yml │\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected output:\n%q\nwant:\n%q", got, want)
	}
}

func TestWriteTableBordered_empty(t *testing.T) {
	var out bytes.Buffer
	if err := WriteTableBordered(&out, nil); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output, got %q", out.String())
	}
}

func TestWriteTableBorderedAll(t *testing.T) {
	rows := [][]string{
		{"ID", "FILE"},
		{"8", "ci.yml"},
	}

	var out bytes.Buffer
	if err := WriteTableBorderedAll(&out, rows); err != nil {
		t.Fatal(err)
	}

	want := "┌────┬────────┐\n" +
		"│ ID │ FILE   │\n" +
		"├────┼────────┤\n" +
		"│ 8  │ ci.yml │\n" +
		"└────┴────────┘\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected output:\n%q\nwant:\n%q", got, want)
	}
}

func TestWriteTableBorderedAll_empty(t *testing.T) {
	var out bytes.Buffer
	if err := WriteTableBorderedAll(&out, nil); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output, got %q", out.String())
	}
}

func TestWriteTableBordered_unicodeWidth(t *testing.T) {
	rows := [][]string{
		{"FILE", "NOTE"},
		{"流水线.yml", "ok"},
		{"ci.yml", "parse error"},
	}

	var out bytes.Buffer
	if err := WriteTableBordered(&out, rows); err != nil {
		t.Fatal(err)
	}

	want := "│ FILE       │ NOTE        │\n" +
		"├────────────┼─────────────┤\n" +
		"│ 流水线.yml │ ok          │\n" +
		"│ ci.yml     │ parse error │\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected output:\n%q\nwant:\n%q", got, want)
	}
}
