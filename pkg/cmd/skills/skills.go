package skills

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/cmdutil"
	skillassets "gitee.com/oschina/gitee-cli/skills"
)

const skillsDirEnv = "AGENTS_SKILLS_DIR"

var legacySkillNames = []string{"gitee-issue-manage", "gitee-repo-manager"}

type skillStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
}

type changeResult struct {
	Directory     string   `json:"directory"`
	Installed     []string `json:"installed,omitempty"`
	Removed       []string `json:"removed,omitempty"`
	RemovedLegacy []string `json:"removed_legacy,omitempty"`
}

// NewSkillsCmd manages the Agent Skills embedded in the current CLI release.
func NewSkillsCmd(f *cmdutil.Factory) *cobra.Command {
	var targetDir string

	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage bundled Agent Skills",
		Long: `List, install, and uninstall the Agent Skills bundled with this CLI release.

Skills are installed offline from the current binary. By default they are
mirrored into ~/.agents/skills without touching unrelated skill directories.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.PersistentFlags().StringVar(&targetDir, "dir", "", "Agent skills directory (default: $AGENTS_SKILLS_DIR or ~/.agents/skills)")
	cmd.AddCommand(newListCmd(f, &targetDir))
	cmd.AddCommand(newInstallCmd(f, &targetDir))
	cmd.AddCommand(newUninstallCmd(f, &targetDir))
	return cmd
}

func newListCmd(f *cmdutil.Factory, targetDir *string) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List bundled Agent Skills",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveTargetDir(*targetDir)
			if err != nil {
				return err
			}
			names, err := skillassets.Names()
			if err != nil {
				return err
			}

			statuses := make([]skillStatus, 0, len(names))
			for _, name := range names {
				path := filepath.Join(dir, name)
				_, err := os.Stat(filepath.Join(path, "SKILL.md"))
				installed := err == nil
				if err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("inspect %s: %w", path, err)
				}
				statuses = append(statuses, skillStatus{Name: name, Installed: installed, Path: path})
			}

			if jsonOut {
				return json.NewEncoder(f.IOStreams.Out).Encode(statuses)
			}
			rows := [][]string{{"SKILL", "STATUS", "PATH"}}
			for _, status := range statuses {
				state := i18n.T("skills.not_installed")
				if status.Installed {
					state = i18n.T("skills.installed")
				}
				rows = append(rows, []string{status.Name, state, status.Path})
			}
			return cmdutil.WriteTable(f.IOStreams.Out, rows)
		},
	}
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func newInstallCmd(f *cmdutil.Factory, targetDir *string) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install or update bundled Agent Skills",
		Long:  `Install the bundled Agent Skills offline, replacing only skill directories managed by Gitee CLI.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveTargetDir(*targetDir)
			if err != nil {
				return err
			}
			result, err := installAll(skillassets.Files, dir)
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(f.IOStreams.Out).Encode(result)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("skills.install_done", len(result.Installed), dir))
			if len(result.RemovedLegacy) > 0 {
				fmt.Fprint(f.IOStreams.Out, i18n.Tf("skills.legacy_removed", len(result.RemovedLegacy)))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func newUninstallCmd(f *cmdutil.Factory, targetDir *string) *cobra.Command {
	var yes bool
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall bundled Agent Skills",
		Long:  `Remove Agent Skills managed by Gitee CLI. Unrelated skill directories are left untouched.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveTargetDir(*targetDir)
			if err != nil {
				return err
			}
			if !yes {
				confirmed, err := cmdutil.ConfirmDestructiveAction(f, i18n.Tf("skills.uninstall_confirm", dir))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(f.IOStreams.Out, i18n.T("aborted"))
					return nil
				}
			}

			result, err := uninstallAll(dir)
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(f.IOStreams.Out).Encode(result)
			}
			fmt.Fprint(f.IOStreams.Out, i18n.Tf("skills.uninstall_done", len(result.Removed), dir))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Output as JSON")
	return cmd
}

func resolveTargetDir(flagValue string) (string, error) {
	if flagValue != "" {
		return filepath.Abs(flagValue)
	}
	if envValue := os.Getenv(skillsDirEnv); envValue != "" {
		return filepath.Abs(envValue)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".agents", "skills"), nil
}

func installAll(source fs.FS, targetDir string) (changeResult, error) {
	result := changeResult{Directory: targetDir}
	names, err := skillassets.Names()
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return result, fmt.Errorf("create skills directory: %w", err)
	}
	for _, name := range names {
		if err := installOne(source, targetDir, name); err != nil {
			return result, fmt.Errorf("install %s: %w", name, err)
		}
		result.Installed = append(result.Installed, name)
	}
	for _, name := range legacySkillNames {
		removed, err := removePath(filepath.Join(targetDir, name))
		if err != nil {
			return result, fmt.Errorf("remove legacy skill %s: %w", name, err)
		}
		if removed {
			result.RemovedLegacy = append(result.RemovedLegacy, name)
		}
	}
	return result, nil
}

func installOne(source fs.FS, targetDir, name string) error {
	tempDir, err := os.MkdirTemp(targetDir, "."+name+"-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := copyEmbeddedDir(source, name, tempDir); err != nil {
		return err
	}

	target := filepath.Join(targetDir, name)
	backup := ""
	if _, err := os.Lstat(target); err == nil {
		backup, err = reservePath(targetDir, "."+name+"-backup-")
		if err != nil {
			return err
		}
		if err := os.Rename(target, backup); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(tempDir, target); err != nil {
		if backup != "" {
			_ = os.Rename(backup, target)
		}
		return err
	}
	if backup != "" {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove previous installation: %w", err)
		}
	}
	return nil
}

func reservePath(parent, pattern string) (string, error) {
	path, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

func copyEmbeddedDir(source fs.FS, sourceDir, targetDir string) error {
	return fs.WalkDir(source, sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(targetDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func uninstallAll(targetDir string) (changeResult, error) {
	result := changeResult{Directory: targetDir}
	names, err := skillassets.Names()
	if err != nil {
		return result, err
	}
	names = append(names, legacySkillNames...)
	sort.Strings(names)
	for _, name := range names {
		removed, err := removePath(filepath.Join(targetDir, name))
		if err != nil {
			return result, fmt.Errorf("remove %s: %w", name, err)
		}
		if removed {
			result.Removed = append(result.Removed, name)
		}
	}
	return result, nil
}

func removePath(path string) (bool, error) {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := os.RemoveAll(path); err != nil {
		return false, err
	}
	return true, nil
}
