package skillassets

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

// Files contains the Agent Skills shipped with this CLI release.
//
//go:embed gitee-*
var Files embed.FS

// Names returns the embedded skill directory names in stable order.
func Names() ([]string, error) {
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded skills: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := fs.Stat(Files, entry.Name()+"/SKILL.md"); err != nil {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}
