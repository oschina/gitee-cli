package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "gitee_1.0.0_darwin_arm64.tar.gz")
	data := []byte("archive contents")
	if err := os.WriteFile(archivePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	checksumPath := filepath.Join(dir, "checksums.txt")
	checksum := fmt.Sprintf("%x  ./%s\n", sha256.Sum256(data), filepath.Base(archivePath))
	if err := os.WriteFile(checksumPath, []byte(checksum), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(archivePath, checksumPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(archivePath, checksumPath); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "release.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	contents := []byte("binary-data")
	if err := tarWriter.WriteHeader(&tar.Header{Name: "gitee", Mode: 0755, Size: int64(len(contents)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(contents); err != nil {
		t.Fatal(err)
	}
	tarWriter.Close()
	gzipWriter.Close()
	file.Close()

	destination := filepath.Join(dir, "gitee")
	if err := extractBinaryFromTarGz(archivePath, "gitee", destination); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, destination, contents)
}

func TestExtractBinaryFromZip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "release.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(file)
	entry, err := zipWriter.Create("gitee.exe")
	if err != nil {
		t.Fatal(err)
	}
	contents := []byte("windows-binary")
	if _, err := entry.Write(contents); err != nil {
		t.Fatal(err)
	}
	zipWriter.Close()
	file.Close()

	destination := filepath.Join(dir, "gitee.exe")
	if err := extractBinaryFromZip(archivePath, "gitee.exe", destination); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, destination, contents)
}

func assertFileContents(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("contents = %q, want %q", got, want)
	}
}
