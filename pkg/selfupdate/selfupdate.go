// Package selfupdate replaces the running llmconfig binary with a newer
// release pulled from GitHub. The package is intentionally narrow: fetch
// release metadata, pick the right asset for the current OS/arch, verify
// SHA256 against the published checksums.txt, and atomically swap the
// binary on disk.
//
// The atomic-swap pattern works on both Unix (where you can overwrite
// running binaries) and Windows (where you can't, but you *can* rename a
// running executable out of the way before placing the new one).
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/httpx"
)

// Repo is the GitHub repository slug llmconfig publishes releases from.
// The selfupdater refuses to download from any other host, so changing
// this is the single point of trust.
const Repo = "kiliczsh/llmconfig"

const releasesAPI = "https://api.github.com/repos/" + Repo + "/releases"
const releasesDownload = "https://github.com/" + Repo + "/releases/download/"

// Release mirrors the subset of GitHub's release JSON that we use.
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Asset is one downloadable file attached to a release.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// LatestRelease fetches the most recent release tagged on the repo.
func LatestRelease() (*Release, error) {
	return fetchRelease(releasesAPI + "/latest")
}

// ReleaseByTag fetches a specific tagged release. Use it for --version pinning.
func ReleaseByTag(tag string) (*Release, error) {
	if tag == "" {
		return nil, errors.New("selfupdate: empty tag")
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return fetchRelease(releasesAPI + "/tags/" + tag)
}

func fetchRelease(url string) (*Release, error) {
	resp, err := httpx.API.Get(url)
	if err != nil {
		return nil, fmt.Errorf("selfupdate: fetch release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("selfupdate: release not found at %s", url)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("selfupdate: GitHub API returned HTTP %d", resp.StatusCode)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("selfupdate: parse release: %w", err)
	}
	return &rel, nil
}

// PickAsset returns the release archive for the current OS/arch. The naming
// convention matches install.sh/install.ps1: llmconfig-<version>-<os>-<arch>.<ext>
// (.zip on Windows, .tar.gz elsewhere).
func PickAsset(rel *Release) (*Asset, error) {
	osName, arch := goosToReleaseOS(runtime.GOOS), goarchToReleaseArch(runtime.GOARCH)
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	want := fmt.Sprintf("-%s-%s%s", osName, arch, ext)

	for i := range rel.Assets {
		a := &rel.Assets[i]
		if strings.HasSuffix(a.Name, want) && strings.HasPrefix(a.Name, "llmconfig-") {
			// Defence in depth: refuse anything that didn't come from our
			// release URL prefix. Catches a malicious asset entry that
			// pointed at a different host.
			if !strings.HasPrefix(a.DownloadURL, releasesDownload) {
				return nil, fmt.Errorf("selfupdate: asset %s has untrusted download URL", a.Name)
			}
			return a, nil
		}
	}
	return nil, fmt.Errorf("selfupdate: no asset for %s/%s in release %s", osName, arch, rel.TagName)
}

func goosToReleaseOS(g string) string {
	switch g {
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	default:
		return g
	}
}

func goarchToReleaseArch(g string) string {
	switch g {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return g
	}
}

// FetchChecksum returns the published SHA256 for an asset, downloaded from
// the release's checksums.txt. Returns ErrNoChecksum if the file or entry
// isn't present — callers can decide whether to fail or proceed.
var ErrNoChecksum = errors.New("selfupdate: no checksum entry for asset")

func FetchChecksum(rel *Release, assetName string) (string, error) {
	url := releasesDownload + rel.TagName + "/checksums.txt"
	resp, err := httpx.API.Get(url)
	if err != nil {
		return "", fmt.Errorf("selfupdate: fetch checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return "", ErrNoChecksum
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("selfupdate: checksums.txt returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("selfupdate: read checksums: %w", err)
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: "<sha256>  <filename>" (two spaces, sha256sum style).
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Filename can have a leading "*" for binary mode.
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if name == assetName {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", ErrNoChecksum
}

// DownloadAsset streams the asset to dest, verifying SHA256 if expected is
// non-empty. onProgress receives (downloaded, total) byte counts as the
// stream advances. Refuses to download from anywhere outside the release
// URL prefix, even if Asset.DownloadURL was tampered with after PickAsset
// validated it (belt and braces).
func DownloadAsset(asset *Asset, dest string, expectedSHA256 string, onProgress func(downloaded, total int64)) error {
	if !strings.HasPrefix(asset.DownloadURL, releasesDownload) {
		return fmt.Errorf("selfupdate: refusing to download from untrusted host: %s", asset.DownloadURL)
	}

	resp, err := httpx.Download.Get(asset.DownloadURL)
	if err != nil {
		return fmt.Errorf("selfupdate: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("selfupdate: download %s: HTTP %d", asset.Name, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("selfupdate: create dest dir: %w", err)
	}
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("selfupdate: create %s: %w", dest, err)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	total := asset.Size
	var downloaded int64
	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := writer.Write(buf[:n]); werr != nil {
				out.Close()
				return fmt.Errorf("selfupdate: write: %w", werr)
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
			out.Close()
			return fmt.Errorf("selfupdate: read: %w", readErr)
		}
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("selfupdate: close: %w", err)
	}

	if expectedSHA256 != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(got, expectedSHA256) {
			_ = os.Remove(dest)
			return fmt.Errorf("selfupdate: checksum mismatch (expected %s, got %s)", expectedSHA256, got)
		}
	}
	return nil
}

// ExtractedBinaries holds the on-disk paths to the llmconfig and (optional)
// llmc binaries inside a downloaded release archive's staging directory.
type ExtractedBinaries struct {
	Llmconfig string
	Llmc      string // empty when the archive doesn't include the alias
	StageDir  string // caller is responsible for cleanup
}

// ExtractRelease unpacks the downloaded archive into a fresh staging
// directory next to it and returns the absolute paths of the binaries
// found. Cleanup of StageDir is the caller's responsibility — typically
// after a successful Apply().
func ExtractRelease(archivePath string) (*ExtractedBinaries, error) {
	stage, err := os.MkdirTemp(filepath.Dir(archivePath), "llmconfig-update-")
	if err != nil {
		return nil, fmt.Errorf("selfupdate: stage dir: %w", err)
	}

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"):
		if err := untarGz(archivePath, stage); err != nil {
			_ = os.RemoveAll(stage)
			return nil, err
		}
	case strings.HasSuffix(archivePath, ".zip"):
		if err := unzip(archivePath, stage); err != nil {
			_ = os.RemoveAll(stage)
			return nil, err
		}
	default:
		_ = os.RemoveAll(stage)
		return nil, fmt.Errorf("selfupdate: unsupported archive format: %s", archivePath)
	}

	out := &ExtractedBinaries{StageDir: stage}
	llmconfigName := "llmconfig"
	llmcName := "llmc"
	if runtime.GOOS == "windows" {
		llmconfigName += ".exe"
		llmcName += ".exe"
	}

	err = filepath.Walk(stage, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		switch info.Name() {
		case llmconfigName:
			if out.Llmconfig == "" {
				out.Llmconfig = path
			}
		case llmcName:
			if out.Llmc == "" {
				out.Llmc = path
			}
		}
		return nil
	})
	if err != nil {
		_ = os.RemoveAll(stage)
		return nil, fmt.Errorf("selfupdate: scan stage: %w", err)
	}
	if out.Llmconfig == "" {
		_ = os.RemoveAll(stage)
		return nil, fmt.Errorf("selfupdate: archive did not contain %s", llmconfigName)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(out.Llmconfig, 0755)
		if out.Llmc != "" {
			_ = os.Chmod(out.Llmc, 0755)
		}
	}
	return out, nil
}

// Swap replaces targetPath with newPath atomically. On Windows the running
// binary cannot be overwritten, but it can be renamed out of the way; we
// always rename current → "<target>.old" before installing the new one.
// If the second rename fails, the original is restored.
//
// Both paths must live on the same filesystem — selfupdate places the
// staging file in the same directory as the target to satisfy this.
func Swap(targetPath, newPath string) error {
	backup := targetPath + ".old"

	// Best-effort cleanup of any leftover .old from a prior update; OK to
	// fail (Windows may keep it locked while another process holds it).
	_ = os.Remove(backup)

	if err := os.Rename(targetPath, backup); err != nil {
		return fmt.Errorf("selfupdate: backup %s: %w", targetPath, err)
	}
	if err := os.Rename(newPath, targetPath); err != nil {
		// Roll back: put the original back where it was.
		_ = os.Rename(backup, targetPath)
		return fmt.Errorf("selfupdate: install %s: %w", targetPath, err)
	}
	return nil
}

func unzip(archivePath, dest string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("selfupdate: open zip: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		clean := filepath.Clean(f.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("selfupdate: zip entry escapes stage: %s", f.Name)
		}
		out := filepath.Join(dest, clean)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(out, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.Create(out)
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(w, rc); err != nil {
			w.Close()
			rc.Close()
			return err
		}
		w.Close()
		rc.Close()
	}
	return nil
}

func untarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("selfupdate: open tar: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("selfupdate: gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("selfupdate: tar entry: %w", err)
		}
		// Reject path traversal — release archives are flat (binary +
		// templates dir), so any "../" or absolute path is malicious.
		clean := filepath.Clean(hdr.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("selfupdate: tar entry escapes stage: %s", hdr.Name)
		}
		out := filepath.Join(dest, clean)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(out, 0755); err != nil {
				return err
			}
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}
		w, err := os.Create(out)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, tr); err != nil {
			w.Close()
			return err
		}
		w.Close()
	}
	return nil
}
