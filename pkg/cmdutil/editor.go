package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gitee.com/oschina/gitee-cli/internal/config"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

func OpenEditor(ios *iostreams.IOStreams, filename, initial string) (string, error) {
	f, err := os.CreateTemp("", filename)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if initial != "" {
		if _, err := f.WriteString(initial); err != nil {
			f.Close()
			return "", fmt.Errorf("write temp file: %w", err)
		}
	}
	f.Close()

	editor := config.Editor()
	parts := strings.Fields(editor)
	args := append(parts[1:], f.Name())
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = ios.In
	cmd.Stdout = ios.Out
	cmd.Stderr = ios.ErrOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", fmt.Errorf("read temp file: %w", err)
	}
	return strings.TrimRight(string(data), "\n"), nil
}
