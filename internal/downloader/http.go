package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type httpDownloader struct{}

func (d *httpDownloader) Download(ctx context.Context, req *Request, onProgress func(downloaded, total int64)) (string, error) {
	if err := os.MkdirAll(req.CacheDir, 0755); err != nil {
		return "", fmt.Errorf("downloader: create cache dir: %w", err)
	}

	destPath := filepath.Join(req.CacheDir, req.File)

	// Resolve URL
	rawURL := req.URL
	if rawURL == "" {
		rawURL = hfResolveURL(req.Repo, req.File)
	}

	// Check existing file size for resume
	var resumeFrom int64
	if req.Resume {
		if info, err := os.Stat(destPath); err == nil {
			resumeFrom = info.Size()
		}
	}

	// HEAD request to get total size and check range support
	headReq, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return "", err
	}
	addAuthHeader(headReq, req.Token)

	headResp, err := http.DefaultClient.Do(headReq)
	if err != nil {
		return "", fmt.Errorf("downloader: HEAD %s: %w", rawURL, err)
	}
	headResp.Body.Close()

	if headResp.StatusCode == 401 {
		return "", fmt.Errorf("downloader: unauthorized — set HUGGINGFACE_TOKEN or use --token")
	}
	if headResp.StatusCode >= 400 {
		return "", fmt.Errorf("downloader: HEAD %s: HTTP %d", rawURL, headResp.StatusCode)
	}

	total := headResp.ContentLength
	acceptsRange := strings.EqualFold(headResp.Header.Get("Accept-Ranges"), "bytes")

	// If file is already complete, skip
	if resumeFrom > 0 && total > 0 && resumeFrom >= total {
		if onProgress != nil {
			onProgress(total, total)
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

	resp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return "", fmt.Errorf("downloader: GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("downloader: GET %s: HTTP %d", rawURL, resp.StatusCode)
	}

	f, err := os.OpenFile(destPath, openFlag, 0644)
	if err != nil {
		return "", fmt.Errorf("downloader: open dest file: %w", err)
	}
	defer f.Close()

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

	if req.VerifyChecksum && req.Checksum != "" {
		if err := verifyChecksum(destPath, req.Checksum); err != nil {
			return "", err
		}
	}

	return destPath, nil
}

func addAuthHeader(req *http.Request, token string) {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}
