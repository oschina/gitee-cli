package update

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gitee.com/oschina/gitee-cli/internal/build"
)

type InstallMethod string

const (
	InstallNPM        InstallMethod = "npm"
	InstallHomebrew   InstallMethod = "homebrew"
	InstallStandalone InstallMethod = "standalone"
)

type Installation struct {
	Method     InstallMethod
	Executable string
	Global     bool
	ScopeKnown bool
	Detail     string
}

type InstallResult struct {
	Deferred bool
}

var commandContext = exec.CommandContext

func DetectInstallation(ctx context.Context) (Installation, error) {
	executable, err := os.Executable()
	if err != nil {
		return Installation{}, fmt.Errorf("resolve current executable: %w", err)
	}
	resolved := executable
	if path, err := filepath.EvalSymlinks(executable); err == nil {
		resolved = path
	}

	distribution := strings.ToLower(strings.TrimSpace(os.Getenv("GITEE_INSTALL_METHOD")))
	if distribution == "" || distribution == "unknown" {
		distribution = strings.ToLower(strings.TrimSpace(build.Distribution))
	}

	if distribution == "npm" || strings.Contains(filepath.ToSlash(resolved), "/node_modules/@gitee/") {
		installation := Installation{Method: InstallNPM, Executable: resolved, Detail: "npm"}
		root, err := npmGlobalRoot(ctx)
		if err == nil {
			installation.ScopeKnown = true
			packageRoot := os.Getenv("GITEE_NPM_PACKAGE_ROOT")
			installation.Global = pathWithin(root, resolved) || (packageRoot != "" && pathWithin(root, packageRoot))
			if installation.Global {
				installation.Detail = "npm global"
			} else {
				installation.Detail = "npm local"
			}
		}
		return installation, nil
	}

	normalizedPath := filepath.ToSlash(resolved)
	if distribution == "homebrew" || strings.Contains(normalizedPath, "/Cellar/gitee-cli/") {
		return Installation{Method: InstallHomebrew, Executable: resolved, Global: true, Detail: "Homebrew"}, nil
	}

	detail := "standalone binary"
	if distribution == "source" {
		detail = "source-installed binary"
	} else if distribution == "release" {
		detail = "Gitee Release binary"
	}
	return Installation{Method: InstallStandalone, Executable: resolved, Global: true, Detail: detail}, nil
}

func (i Installation) Description() string {
	if i.Detail != "" {
		return i.Detail
	}
	return string(i.Method)
}

func ApplyUpdate(ctx context.Context, release *ReleaseInfo, installation Installation, stdout, stderr io.Writer) (InstallResult, error) {
	if release == nil {
		return InstallResult{}, fmt.Errorf("release information is required")
	}
	switch installation.Method {
	case InstallNPM:
		if !installation.ScopeKnown {
			return InstallResult{}, fmt.Errorf("could not verify whether this npm installation is global; update it manually with npm install -g @gitee/gitee-cli@%s", strings.TrimPrefix(release.Version, "v"))
		}
		if !installation.Global {
			return InstallResult{}, fmt.Errorf("this is a local npm installation; update it from the owning project with npm install @gitee/gitee-cli@%s", strings.TrimPrefix(release.Version, "v"))
		}
		cmd := commandContext(ctx, "npm", "install", "--global", "@gitee/gitee-cli@"+strings.TrimPrefix(release.Version, "v"))
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return InstallResult{}, fmt.Errorf("npm update failed: %w", err)
		}
		return InstallResult{}, nil
	case InstallHomebrew:
		cmd := commandContext(ctx, "brew", "upgrade", "gitee-cli")
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return InstallResult{}, fmt.Errorf("Homebrew update failed: %w", err)
		}
		return InstallResult{}, nil
	case InstallStandalone:
		deferred, err := installStandalone(ctx, release, installation.Executable)
		return InstallResult{Deferred: deferred}, err
	default:
		return InstallResult{}, fmt.Errorf("unsupported installation method: %s", installation.Method)
	}
}

func npmGlobalRoot(ctx context.Context) (string, error) {
	output, err := commandContext(ctx, "npm", "root", "--global").Output()
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", fmt.Errorf("npm returned an empty global root")
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return filepath.Clean(root), nil
}

func pathWithin(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func releaseArchiveName(version string) string {
	extension := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".zip"
	}
	return fmt.Sprintf("gitee_%s_%s_%s%s", strings.TrimPrefix(version, "v"), runtime.GOOS, runtime.GOARCH, extension)
}
