package llamacpp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const githubReleasesURL = "https://api.github.com/repos/ggml-org/llama.cpp/releases/latest"

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GithubAsset `json:"assets"`
}

type GithubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// LatestRelease fetches the latest llama.cpp release metadata from GitHub.
func LatestRelease() (*githubRelease, error) {
	resp, err := http.Get(githubReleasesURL)
	if err != nil {
		return nil, fmt.Errorf("install: fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("install: GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("install: parse release: %w", err)
	}
	return &rel, nil
}

// PickAsset selects the best release asset for the current OS/arch/backend.
// backend: "cuda", "metal", "cpu", "" (auto-detect)
func PickAsset(rel *githubRelease, backend string) (*GithubAsset, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Build preference list
	var patterns []string
	switch goos {
	case "windows":
		switch backend {
		case "cuda":
			patterns = []string{"win-cuda", "windows-cuda"}
		case "cpu":
			patterns = []string{"win-avx2", "win-avx", "windows-avx2", "win-cpu"}
		default:
			// Auto: prefer CUDA, fallback to AVX2
			patterns = []string{"win-cuda", "win-avx2", "win-avx", "windows-avx2", "win-cpu"}
		}
	case "darwin":
		if goarch == "arm64" {
			patterns = []string{"macos-arm64", "osx-arm64"}
		} else {
			patterns = []string{"macos-x64", "osx-x64", "macos-x86_64"}
		}
	case "linux":
		switch backend {
		case "cuda":
			patterns = []string{"linux-cuda", "ubuntu-cuda"}
		case "cpu":
			patterns = []string{"linux-x64", "linux-avx2"}
		default:
			patterns = []string{"linux-cuda", "ubuntu-cuda", "linux-x64", "linux-avx2"}
		}
	}

	lower := func(s string) string { return strings.ToLower(s) }

	for _, pattern := range patterns {
		for i := range rel.Assets {
			a := &rel.Assets[i]
			name := lower(a.Name)
			if strings.Contains(name, pattern) && (strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz")) {
				return a, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable asset found for %s/%s (backend: %q)\nAvailable assets:\n%s",
		goos, goarch, backend, listAssets(rel))
}

func listAssets(rel *githubRelease) string {
	var sb strings.Builder
	for _, a := range rel.Assets {
		if strings.HasSuffix(a.Name, ".zip") || strings.HasSuffix(a.Name, ".tar.gz") {
			fmt.Fprintf(&sb, "  %s\n", a.Name)
		}
	}
	return sb.String()
}

// Install downloads and extracts a release asset into BinDir().
// onProgress is called with (downloaded, total) bytes.
func Install(asset *GithubAsset, onProgress func(downloaded, total int64)) error {
	if err := os.MkdirAll(BinDir(), 0755); err != nil {
		return fmt.Errorf("install: create bin dir: %w", err)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "llama-*")
	if err != nil {
		return fmt.Errorf("install: temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("install: download: %w", err)
	}
	defer resp.Body.Close()

	total := asset.Size
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("install: write: %w", writeErr)
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("install: read: %w", readErr)
		}
	}
	tmpFile.Close()

	// Extract
	if strings.HasSuffix(asset.Name, ".zip") {
		return extractZip(tmpFile.Name(), BinDir())
	}
	if strings.HasSuffix(asset.Name, ".tar.gz") {
		return extractTarGz(tmpFile.Name(), BinDir())
	}
	return fmt.Errorf("install: unsupported archive format: %s", asset.Name)
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("extract: open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		// Only extract binaries and DLLs we care about
		if !isUsefulFile(name) {
			continue
		}

		destPath := filepath.Join(destDir, name)
		if err := extractZipFile(f, destPath); err != nil {
			return fmt.Errorf("extract: %s: %w", name, err)
		}

		// Make executable on Unix
		if runtime.GOOS != "windows" {
			_ = os.Chmod(destPath, 0755)
		}
	}
	return nil
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
		destPath := filepath.Join(destDir, name)

		switch hdr.Typeflag {
		case tar.TypeSymlink:
			if !isUsefulFile(name) {
				continue
			}
			_ = os.Remove(destPath)
			if err := os.Symlink(filepath.Base(hdr.Linkname), destPath); err != nil {
				return fmt.Errorf("extract: symlink %s: %w", name, err)
			}
		case tar.TypeReg:
			if !isUsefulFile(name) {
				continue
			}
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
			}
		}
	}
	return nil
}

func isUsefulFile(name string) bool {
	lower := strings.ToLower(name)
	useful := []string{"llama-server", "llama-cli", "llama-server.exe", "llama-cli.exe"}
	for _, u := range useful {
		if lower == u {
			return true
		}
	}
	// Include DLLs and .so files for backends
	return strings.HasSuffix(lower, ".dll") || strings.HasSuffix(lower, ".so") || strings.HasSuffix(lower, ".dylib")
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
