package ikllamacpp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Extract installs ik_llama.cpp from a local archive. Mirrors
// pkg/llamacpp.Extract: only files matching isUsefulFile() are copied into
// BinDir(), so users can safely point --file at a full release archive
// without polluting the bin dir.
func Extract(archivePath string) error {
	if err := os.MkdirAll(BinDir(), 0755); err != nil {
		return fmt.Errorf("install: create bin dir: %w", err)
	}
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, BinDir())
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, BinDir())
	default:
		return fmt.Errorf("unsupported archive format: %s (want .zip or .tar.gz)", archivePath)
	}
}

func clearQuarantine(path string) {
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-d", "com.apple.quarantine", path).Run()
	}
}

func isUsefulFile(name string) bool {
	lower := strings.ToLower(name)
	useful := []string{"llama-server", "llama-cli", "llama-server.exe", "llama-cli.exe"}
	for _, u := range useful {
		if lower == u {
			return true
		}
	}
	return strings.HasSuffix(lower, ".dll") || strings.HasSuffix(lower, ".so") || strings.HasSuffix(lower, ".dylib")
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("extract: open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if !isUsefulFile(name) {
			continue
		}
		destPath := filepath.Join(destDir, name)
		if err := extractZipFile(f, destPath); err != nil {
			return fmt.Errorf("extract: %s: %w", name, err)
		}
		if runtime.GOOS != "windows" {
			_ = os.Chmod(destPath, 0755)
			clearQuarantine(destPath)
		}
	}
	return nil
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

func extractTarGz(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("extract: open tar.gz: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("extract: gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("extract: read tar: %w", err)
		}
		name := filepath.Base(hdr.Name)
		if !isUsefulFile(name) {
			continue
		}
		destPath := filepath.Join(destDir, name)

		switch hdr.Typeflag {
		case tar.TypeSymlink:
			_ = os.Remove(destPath)
			if err := os.Symlink(filepath.Base(hdr.Linkname), destPath); err != nil {
				return fmt.Errorf("extract: symlink %s: %w", name, err)
			}
		case tar.TypeReg:
			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("extract: create %s: %w", name, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("extract: write %s: %w", name, err)
			}
			out.Close()
			if runtime.GOOS != "windows" {
				_ = os.Chmod(destPath, 0755)
				clearQuarantine(destPath)
			}
		}
	}
	return nil
}
