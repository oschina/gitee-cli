//go:build windows

package update

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func replaceExecutable(currentPath, newBinaryPath string) (bool, error) {
	dir := filepath.Dir(currentPath)
	helper, err := os.CreateTemp(dir, ".gitee-update-*.exe")
	if err != nil {
		return false, err
	}
	helperPath := helper.Name()
	source, err := os.Open(newBinaryPath)
	if err != nil {
		helper.Close()
		os.Remove(helperPath)
		return false, err
	}
	_, copyErr := io.Copy(helper, source)
	source.Close()
	closeErr := helper.Close()
	if copyErr != nil {
		os.Remove(helperPath)
		return false, copyErr
	}
	if closeErr != nil {
		os.Remove(helperPath)
		return false, closeErr
	}

	cmd := exec.Command(helperPath, "update", "--apply", currentPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200}
	if err := cmd.Start(); err != nil {
		os.Remove(helperPath)
		return false, err
	}
	return true, nil
}

func ApplyPendingUpdate(targetPath string) error {
	helperPath, err := os.Executable()
	if err != nil {
		return err
	}
	backup, err := os.CreateTemp(filepath.Dir(targetPath), ".gitee-backup-*.exe")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	backup.Close()
	os.Remove(backupPath)
	deadline := time.Now().Add(30 * time.Second)
	for {
		_ = os.Remove(backupPath)
		err = os.Rename(targetPath, backupPath)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting to replace %s: %w", targetPath, err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	source, err := os.Open(helperPath)
	if err != nil {
		_ = os.Rename(backupPath, targetPath)
		return err
	}
	destination, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		source.Close()
		_ = os.Rename(backupPath, targetPath)
		return err
	}
	_, copyErr := io.Copy(destination, source)
	source.Close()
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(targetPath)
		_ = os.Rename(backupPath, targetPath)
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	_ = os.Remove(backupPath)

	cleanup := exec.Command("cmd.exe", "/C", "ping 127.0.0.1 -n 2 >NUL & del /F /Q \""+helperPath+"\"")
	cleanup.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200}
	_ = cleanup.Start()
	return nil
}
