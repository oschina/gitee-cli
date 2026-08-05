package tui

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		input    string
		maxWidth int
		want     string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"hello", 5, "hello"},
		{"hi", 3, "hi"},
		{"hello", 3, "hel"},
		{"hello", 0, ""},
		{"中文测试字符", 6, "中..."},
		{"short", 100, "short"},
	}
	for _, tc := range cases {
		got := Truncate(tc.input, tc.maxWidth)
		if got != tc.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.input, tc.maxWidth, got, tc.want)
		}
	}
}
