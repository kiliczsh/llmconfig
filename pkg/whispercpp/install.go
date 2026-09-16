package whispercpp

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/httpx"
)

func clearQuarantine(path string) {
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-d", "com.apple.quarantine", path).Run()
	}
}

const githubReleasesURL = "https://api.github.com/repos/ggml-org/whisper.cpp/releases"

type GithubRelease struct {
	TagName string        `json:"tag_name"`
	Body    string        `json:"body"`
	Assets  []GithubAsset `json:"assets"`
}

type GithubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// LatestRelease fetches the latest whisper.cpp release metadata from GitHub.
func LatestRelease() (*GithubRelease, error) {
	return latestRelease(githubReleasesURL)
}

func latestRelease(releasesURL string) (*GithubRelease, error) {
	rel, err := fetchRelease(releasesURL + "/latest")
	if err != nil {
		return nil, err
	}
	return resolveBuildRelease(releasesURL, rel)
}

var nightlyBuildPattern = regexp.MustCompile(`(?i)nightly build:\**\s*\[?(b[0-9]+)`)

func resolveBuildRelease(releasesURL string, rel *GithubRelease) (*GithubRelease, error) {
	if len(rel.Assets) > 0 {
		return rel, nil
	}

	match := nightlyBuildPattern.FindStringSubmatch(rel.Body)
	if len(match) != 2 {
		return nil, fmt.Errorf("install: release %s has no binary assets or nightly build pointer", rel.TagName)
	}

	buildTag := match[1]
	build, err := releaseByTag(releasesURL, buildTag)
	if err != nil {
		return nil, fmt.Errorf("install: resolve stable release %s build %s: %w", rel.TagName, buildTag, err)
	}
	// Keep the stable version in user-facing output while sourcing its
	// platform archives from the associated build release.
	rel.Assets = build.Assets
	return rel, nil
}

func fetchRelease(releaseURL string) (*GithubRelease, error) {
	resp, err := httpx.API.Get(releaseURL)
	if err != nil {
		return nil, fmt.Errorf("install: fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("install: GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("install: parse release: %w", err)
	}
	return &rel, nil
}

// ReleaseByTag fetches release metadata for a specific whisper.cpp tag from GitHub.
func ReleaseByTag(tag string) (*GithubRelease, error) {
	rel, err := releaseByTag(githubReleasesURL, tag)
	if err != nil {
		return nil, err
	}
	return resolveBuildRelease(githubReleasesURL, rel)
}

func releaseByTag(releasesURL, tag string) (*GithubRelease, error) {
	if strings.ContainsAny(tag, "/\\?#") {
		return nil, fmt.Errorf("install: invalid release tag %q", tag)
	}
	return fetchRelease(releasesURL + "/tags/" + tag)
}

// PickAsset selects the best release asset for the current OS/arch/backend.
// backend: "cuda", "cpu", "" (auto-detect)
// Note: Linux and macOS do not have pre-built CLI binaries — returns an error.
func PickAsset(rel *GithubRelease, backend string) (*GithubAsset, error) {
	goos := runtime.GOOS

	if goos != "windows" {
		return nil, fmt.Errorf("pre-built whisper.cpp binaries are only available for Windows — build from source on %s: https://github.com/ggml-org/whisper.cpp", goos)
	}

	// Windows: prefer CUDA, fall back to CPU
	var patterns []string
	switch backend {
	case "cuda":
		patterns = []string{"cublas-12", "cublas-11", "cublas"}
	case "cpu":
		patterns = []string{"whisper-bin-x64", "blas-bin-x64"}
	default:
		patterns = []string{"cublas-12", "cublas-11", "cublas", "whisper-bin-x64", "blas-bin-x64"}
	}

	lower := func(s string) string { return strings.ToLower(s) }

	for _, pattern := range patterns {
		for i := range rel.Assets {
			a := &rel.Assets[i]
			name := lower(a.Name)
			if strings.Contains(name, pattern) && strings.HasSuffix(name, ".zip") {
				return a, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable asset found for %s (backend: %q)\nAvailable assets:\n%s",
		goos, backend, listAssets(rel))
}

func listAssets(rel *GithubRelease) string {
	var sb strings.Builder
	for _, a := range rel.Assets {
		if strings.HasSuffix(a.Name, ".zip") {
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

	tmpFile, err := os.CreateTemp("", "whisper-cpp-*")
	if err != nil {
		return fmt.Errorf("install: temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := httpx.Download.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("install: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("install: download %s: HTTP %d", asset.Name, resp.StatusCode)
	}

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

	if strings.HasSuffix(asset.Name, ".zip") {
		return extractZip(tmpFile.Name(), BinDir())
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

// Extract installs whisper.cpp from a local zip file.
func Extract(archivePath string) error {
	if err := os.MkdirAll(BinDir(), 0755); err != nil {
		return fmt.Errorf("install: create bin dir: %w", err)
	}
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return extractZip(archivePath, BinDir())
	}
	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

func isUsefulFile(name string) bool {
	lower := strings.ToLower(name)
	useful := []string{"whisper-cli", "whisper-cli.exe", "whisper-server", "whisper-server.exe"}
	for _, u := range useful {
		if lower == u {
			return true
		}
	}
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
