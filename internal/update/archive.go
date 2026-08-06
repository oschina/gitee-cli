package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const maxArchiveSize = 512 << 20

func installStandalone(ctx context.Context, release *ReleaseInfo, executable string) (bool, error) {
	archiveName := releaseArchiveName(release.Version)
	archiveAsset, ok := findAsset(release.Assets, archiveName)
	if !ok {
		return false, fmt.Errorf("release %s does not contain %s", release.Version, archiveName)
	}
	checksumsAsset, ok := findAsset(release.Assets, "checksums.txt")
	if !ok {
		return false, fmt.Errorf("release %s does not contain checksums.txt", release.Version)
	}

	tempDir, err := os.MkdirTemp("", "gitee-update-*")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, archiveName)
	checksumPath := filepath.Join(tempDir, "checksums.txt")
	if err := downloadFile(ctx, archiveAsset.BrowserDownloadURL, archivePath, maxArchiveSize); err != nil {
		return false, fmt.Errorf("download %s: %w", archiveName, err)
	}
	if err := downloadFile(ctx, checksumsAsset.BrowserDownloadURL, checksumPath, 1<<20); err != nil {
		return false, fmt.Errorf("download checksums.txt: %w", err)
	}
	if err := verifyChecksum(archivePath, checksumPath); err != nil {
		return false, err
	}

	binaryName := "gitee"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	stagedBinary := filepath.Join(tempDir, binaryName)
	if strings.HasSuffix(archiveName, ".zip") {
		err = extractBinaryFromZip(archivePath, binaryName, stagedBinary)
	} else {
		err = extractBinaryFromTarGz(archivePath, binaryName, stagedBinary)
	}
	if err != nil {
		return false, err
	}
	if err := verifyBinaryVersion(ctx, stagedBinary, release.Version); err != nil {
		return false, err
	}

	deferred, err := replaceExecutable(executable, stagedBinary)
	if err != nil {
		return false, fmt.Errorf("replace %s: %w", executable, err)
	}
	return deferred, nil
}

func findAsset(assets []ReleaseAsset, name string) (ReleaseAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name && asset.BrowserDownloadURL != "" {
			return asset, true
		}
	}
	return ReleaseAsset{}, false
}

func downloadFile(ctx context.Context, sourceURL, destination string, maxSize int64) error {
	req, err := newUpdateRequest(sourceURL)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	if resp.ContentLength > maxSize {
		return fmt.Errorf("download is too large: %d bytes", resp.ContentLength)
	}

	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(resp.Body, maxSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxSize {
		return fmt.Errorf("download exceeds %d bytes", maxSize)
	}
	return nil
}

func verifyChecksum(archivePath, checksumPath string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	archiveName := filepath.Base(archivePath)
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if filepath.Base(name) == archiveName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if len(expected) != sha256.Size*2 {
		return fmt.Errorf("checksums.txt does not contain a valid SHA-256 for %s", archiveName)
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("invalid SHA-256 for %s", archiveName)
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s", archiveName)
	}
	return nil
}

func extractBinaryFromTarGz(archivePath, binaryName, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}
		if header.Size < 0 || header.Size > maxArchiveSize {
			return fmt.Errorf("binary in archive has invalid size")
		}
		return writeExtractedBinary(destination, io.LimitReader(tarReader, header.Size))
	}
	return fmt.Errorf("archive does not contain %s", binaryName)
}

func extractBinaryFromZip(archivePath, binaryName, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || filepath.Base(file.Name) != binaryName {
			continue
		}
		if file.UncompressedSize64 > maxArchiveSize {
			return fmt.Errorf("binary in archive is too large")
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		err = writeExtractedBinary(destination, io.LimitReader(source, maxArchiveSize+1))
		closeErr := source.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	return fmt.Errorf("archive does not contain %s", binaryName)
}

func writeExtractedBinary(destination string, source io.Reader) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, source)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func verifyBinaryVersion(ctx context.Context, binaryPath, expectedVersion string) error {
	output, err := exec.CommandContext(ctx, binaryPath, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify downloaded binary: %w", err)
	}
	if !strings.Contains(string(output), expectedVersion) {
		return fmt.Errorf("downloaded binary reports an unexpected version: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
