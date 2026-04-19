package downloader

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const hfBaseURL = "https://huggingface.co"

func hfResolveURL(repo, file string) string {
	return fmt.Sprintf("%s/%s/resolve/main/%s", hfBaseURL, repo, file)
}

type hfFileMeta struct {
	Filename string `json:"rfilename"`
	Size     int64  `json:"size"`
}

// ListRepoFiles returns all files in a HuggingFace repo.
func ListRepoFiles(repo, token string) ([]hfFileMeta, error) {
	url := fmt.Sprintf("https://huggingface.co/api/models/%s", repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("huggingface: list %s: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("huggingface: unauthorized — set HUGGINGFACE_TOKEN or use --token")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("huggingface: list %s: HTTP %d", repo, resp.StatusCode)
	}

	var meta struct {
		Siblings []hfFileMeta `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("huggingface: parse response: %w", err)
	}
	return meta.Siblings, nil
}

// FindGGUF finds a GGUF file in the repo matching the given quant string.
// Returns (filename, size, error).
func FindGGUF(repo, quant, token string) (string, int64, error) {
	files, err := ListRepoFiles(repo, token)
	if err != nil {
		return "", 0, err
	}

	quantUpper := strings.ToUpper(quant)
	for _, f := range files {
		if !strings.HasSuffix(f.Filename, ".gguf") {
			continue
		}
		if strings.Contains(strings.ToUpper(f.Filename), quantUpper) {
			return f.Filename, f.Size, nil
		}
	}

	// Collect available quants for a helpful error
	var available []string
	for _, f := range files {
		if strings.HasSuffix(f.Filename, ".gguf") {
			available = append(available, f.Filename)
		}
	}
	if len(available) > 0 {
		return "", 0, fmt.Errorf("no GGUF file matching %q in %s\navailable: %s",
			quant, repo, strings.Join(available, "\n  "))
	}
	return "", 0, fmt.Errorf("no GGUF files found in %s", repo)
}
