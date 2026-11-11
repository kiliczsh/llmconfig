package prismllamacpp

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/httpx"
	"github.com/kiliczsh/llmconfig/pkg/llamacpp"
)

const releasesURL = "https://api.github.com/repos/PrismML-Eng/llama.cpp/releases"

// RecommendedReleaseTag tracks the binary release pinned by Prism ML's
// Bonsai demo. Their newest GitHub release can temporarily contain only
// runtime assets while platform archives are still being published.
const RecommendedReleaseTag = "prism-b10683-d8f26ee"

type GithubRelease = llamacpp.GithubRelease
type GithubAsset = llamacpp.GithubAsset

func LatestRelease() (*GithubRelease, error) {
	return fetchRelease(releasesURL + "/latest")
}

func ReleaseByTag(tag string) (*GithubRelease, error) {
	if strings.ContainsAny(tag, "/\\?#") {
		return nil, fmt.Errorf("install: invalid release tag %q", tag)
	}
	return fetchRelease(releasesURL + "/tags/" + tag)
}

func fetchRelease(url string) (*GithubRelease, error) {
	resp, err := httpx.API.Get(url)
	if err != nil {
		return nil, fmt.Errorf("install: fetch Prism release info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("install: GitHub API returned HTTP %d", resp.StatusCode)
	}
	var rel GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("install: parse Prism release: %w", err)
	}
	return &rel, nil
}

// PickAssets returns the platform archive followed by any required runtime
// archive. Prism's Windows CUDA release ships cudart separately.
func PickAssets(rel *GithubRelease, backend string) ([]*GithubAsset, error) {
	return pickAssets(rel, backend, runtime.GOOS, runtime.GOARCH)
}

func pickAssets(rel *GithubRelease, backend, goos, goarch string) ([]*GithubAsset, error) {
	arch := goarch
	if goarch == "amd64" {
		arch = "x64"
	}
	var patterns []string
	switch goos {
	case "windows":
		switch backend {
		case "cuda":
			patterns = []string{"bin-win-cuda"}
		case "cpu":
			patterns = []string{"bin-win-cpu", "bin-win-avx"}
		default:
			patterns = []string{"bin-win-cuda", "bin-win-cpu", "bin-win-avx"}
		}
	case "darwin":
		patterns = []string{"bin-macos-"}
	case "linux":
		if backend == "cuda" {
			patterns = []string{"bin-ubuntu-cuda", "bin-linux-cuda"}
		} else if backend == "cpu" {
			patterns = []string{"bin-ubuntu-" + arch, "bin-linux-" + arch}
		} else {
			patterns = []string{"bin-ubuntu-cuda", "bin-linux-cuda", "bin-ubuntu-" + arch, "bin-linux-" + arch}
		}
	}

	var main *GithubAsset
	for _, pattern := range patterns {
		for i := range rel.Assets {
			asset := &rel.Assets[i]
			name := strings.ToLower(asset.Name)
			if strings.HasPrefix(name, "cudart-") {
				continue
			}
			stem := strings.TrimSuffix(strings.TrimSuffix(name, ".tar.gz"), ".zip")
			stem = strings.TrimSuffix(stem, "-kleidiai")
			if strings.Contains(stem, pattern) && strings.HasSuffix(stem, "-"+arch) && isArchive(name) {
				main = asset
				break
			}
		}
		if main != nil {
			break
		}
	}
	if main == nil {
		var names []string
		for i := range rel.Assets {
			names = append(names, rel.Assets[i].Name)
		}
		return nil, fmt.Errorf("no suitable Prism llama.cpp asset for %s/%s (backend: %q); available: %s", goos, goarch, backend, strings.Join(names, ", "))
	}

	assets := []*GithubAsset{main}
	if goos == "windows" && strings.Contains(strings.ToLower(main.Name), "cuda") {
		mainName := strings.ToLower(main.Name)
		cudaSuffix := ""
		if idx := strings.Index(mainName, "bin-win-cuda-"); idx >= 0 {
			cudaSuffix = mainName[idx:]
		}
		for i := range rel.Assets {
			asset := &rel.Assets[i]
			name := strings.ToLower(asset.Name)
			if strings.HasPrefix(name, "cudart-") && isArchive(name) && (cudaSuffix == "" || strings.Contains(name, cudaSuffix)) {
				assets = append(assets, asset)
				break
			}
		}
	}
	return assets, nil
}

func isArchive(name string) bool {
	return strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz")
}

func Install(asset *GithubAsset, onProgress func(downloaded, total int64)) error {
	return llamacpp.InstallTo(asset, BinDir(), onProgress)
}

func Extract(archivePath string) error {
	return llamacpp.ExtractTo(archivePath, BinDir())
}
