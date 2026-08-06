//go:build !windows

package update

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func replaceExecutable(currentPath, newBinaryPath string) (bool, error) {
	currentInfo, err := os.Stat(currentPath)
	if err != nil {
		return false, err
	}
	dir := filepath.Dir(currentPath)
	staged, err := os.CreateTemp(dir, ".gitee-update-*")
	if err != nil {
		return false, fmt.Errorf("create update file: %w", err)
	}
	stagedPath := staged.Name()
	defer os.Remove(stagedPath)

	source, err := os.Open(newBinaryPath)
	if err != nil {
		staged.Close()
		return false, err
	}
	_, copyErr := io.Copy(staged, source)
	sourceCloseErr := source.Close()
	if copyErr == nil {
		copyErr = sourceCloseErr
	}
	if copyErr == nil {
		copyErr = staged.Chmod(currentInfo.Mode().Perm())
	}
	if copyErr == nil {
		copyErr = staged.Sync()
	}
	closeErr := staged.Close()
	if copyErr != nil {
		return false, copyErr
	}
	if closeErr != nil {
		return false, closeErr
	}

	backup, err := os.CreateTemp(dir, ".gitee-backup-*")
	if err != nil {
		return false, err
	}
	backupPath := backup.Name()
	backup.Close()
	os.Remove(backupPath)
	if err := os.Rename(currentPath, backupPath); err != nil {
		return false, err
	}
	if err := os.Rename(stagedPath, currentPath); err != nil {
		_ = os.Rename(backupPath, currentPath)
		return false, err
	}
	_ = os.Remove(backupPath)
	return false, nil
}

func ApplyPendingUpdate(string) error {
	return fmt.Errorf("deferred updates are only supported on Windows")
}
