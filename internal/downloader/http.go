package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/httpx"
)

type httpDownloader struct{}

func (d *httpDownloader) Download(ctx context.Context, req *Request, onProgress func(downloaded, total int64)) (string, error) {
	if err := os.MkdirAll(req.ModelDir, 0755); err != nil {
		return "", fmt.Errorf("downloader: create models dir: %w", err)
	}

	destPath := filepath.Join(req.ModelDir, req.File)
	tmpPath := destPath + ".tmp"
	// req.File can include a subdirectory (HF repos expose files like
	// `subdir/model.gguf`); make sure the parent exists before we open it.
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("downloader: create dest dir: %w", err)
	}

	// Already fully downloaded — nothing to do.
	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}

	// Resolve URL
	rawURL := req.URL
	if rawURL == "" {
		rawURL = hfResolveURL(req.Repo, req.File)
	}

	// Check in-progress temp file size for resume.
	var resumeFrom int64
	if req.Resume {
		if info, err := os.Stat(tmpPath); err == nil {
			resumeFrom = info.Size()
		}
	}

	// HEAD request to get total size and check range support
	headReq, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return "", err
	}
	addAuthHeader(headReq, req.Token)

	headResp, err := httpx.Download.Do(headReq)
	if err != nil {
		return "", fmt.Errorf("downloader: HEAD %s: %w", rawURL, err)
	}
	headResp.Body.Close()

	if headResp.StatusCode == 401 || headResp.StatusCode == 403 {
		// 401/403 usually means one of:
		//   - no/invalid token — `hf auth login` or set HUGGINGFACE_TOKEN
		//   - gated repo whose license you haven't accepted yet — visit
		//     the repo on huggingface.co once and click "Agree and access"
		// Point the user at the exact repo page when the URL shape
		// allows us to derive it.
		hint := "https://huggingface.co/"
		if repoURL := hfRepoPageFromAsset(rawURL); repoURL != "" {
			hint = repoURL
		}
		action := "ensure you have a valid HF token (hf auth login or HUGGINGFACE_TOKEN)"
		if headResp.StatusCode == 403 {
			action = "accept the repo license at " + hint + " (click \"Agree and access repository\"), then retry"
		} else {
			action += "; if the repo is gated, also accept the license at " + hint
		}
		return "", fmt.Errorf("downloader: %s returned HTTP %d — %s",
			rawURL, headResp.StatusCode, action)
	}
	if headResp.StatusCode >= 400 {
		return "", fmt.Errorf("downloader: HEAD %s: HTTP %d", rawURL, headResp.StatusCode)
	}

	total := headResp.ContentLength
	acceptsRange := strings.EqualFold(headResp.Header.Get("Accept-Ranges"), "bytes")

	// Temp file is already complete — just rename and return.
	if resumeFrom > 0 && total > 0 && resumeFrom >= total {
		if onProgress != nil {
			onProgress(total, total)
		}
		if err := os.Rename(tmpPath, destPath); err != nil {
			return "", fmt.Errorf("downloader: rename complete file: %w", err)
		}
		return destPath, nil
	}

	// Build GET request
	getReq, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	addAuthHeader(getReq, req.Token)

	var openFlag int
	if resumeFrom > 0 && acceptsRange {
		getReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
		openFlag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	} else {
		resumeFrom = 0
		openFlag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	resp, err := httpx.Download.Do(getReq)
	if err != nil {
		return "", fmt.Errorf("downloader: GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resumeFrom > 0 && acceptsRange {
		// Prevent corruption when a server ignores Range and sends the full body with 200.
		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
				return "", fmt.Errorf("downloader: remove partial file: %w", err)
			}

			resumeFrom = 0
			getReq, err = http.NewRequestWithContext(ctx, "GET", rawURL, nil)
			if err != nil {
				return "", err
			}
			addAuthHeader(getReq, req.Token)

			resp, err = httpx.Download.Do(getReq)
			if err != nil {
				return "", fmt.Errorf("downloader: GET %s: %w", rawURL, err)
			}
			defer resp.Body.Close()
			openFlag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		} else if resp.StatusCode != http.StatusPartialContent {
			return "", fmt.Errorf("download: unexpected status %d", resp.StatusCode)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("downloader: GET %s: HTTP %d", rawURL, resp.StatusCode)
	}

	f, err := os.OpenFile(tmpPath, openFlag, 0644)
	if err != nil {
		return "", fmt.Errorf("downloader: open dest file: %w", err)
	}

	downloaded := resumeFrom
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return "", fmt.Errorf("downloader: write: %w", writeErr)
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
			return "", fmt.Errorf("downloader: read: %w", readErr)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	// Close before checksum / rename.
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("downloader: close: %w", err)
	}

	if req.VerifyChecksum && req.Checksum != "" {
		if err := verifyChecksum(tmpPath, req.Checksum); err != nil {
			_ = os.Remove(tmpPath)
			return "", err
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", fmt.Errorf("downloader: rename to final path: %w", err)
	}

	return destPath, nil
}

func addAuthHeader(req *http.Request, token string) {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// hfRepoPageFromAsset extracts the repo landing page from an HF asset URL.
// huggingface.co/<owner>/<repo>/resolve/<ref>/<path> → huggingface.co/<owner>/<repo>.
// Returns "" for URLs that don't match the expected shape (non-HF hosts,
// malformed, etc.) so the caller can fall back to a generic hint.
func hfRepoPageFromAsset(rawURL string) string {
	const host = "https://huggingface.co/"
	rest, found := strings.CutPrefix(rawURL, host)
	if !found {
		return ""
	}
	idx := strings.Index(rest, "/resolve/")
	if idx < 0 {
		idx = strings.Index(rest, "/blob/")
	}
	if idx <= 0 {
		return ""
	}
	return host + rest[:idx]
}
