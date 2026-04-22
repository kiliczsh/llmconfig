package state

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kiliczsh/llamaconfig/internal/process"
)

type fileLock struct {
	path string
	f    *os.File
}

func acquireLock(lockPath string) (*fileLock, error) {
	const maxRetries = 20
	const retryDelay = 50 * time.Millisecond
	const staleAge = 30 * time.Second

	for i := 0; i < maxRetries; i++ {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			// Best-effort: record PID so stale-detection on the next holder
			// can check liveness rather than waiting for mtime to age out.
			fmt.Fprintf(f, "%d", os.Getpid())
			return &fileLock{path: lockPath, f: f}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("state lock: %w", err)
		}

		if isStaleLock(lockPath, staleAge) {
			// Removal may race with another acquirer; either way, retry.
			_ = os.Remove(lockPath)
			continue
		}

		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("state lock: could not acquire after %d retries", maxRetries)
}

func isStaleLock(lockPath string, staleAge time.Duration) bool {
	if raw, err := os.ReadFile(lockPath); err == nil {
		if pid, parseErr := strconv.Atoi(strings.TrimSpace(string(raw))); parseErr == nil && pid > 0 {
			if !process.PidAlive(pid) {
				return true
			}
		}
	}
	info, err := os.Stat(lockPath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) > staleAge
}

func (l *fileLock) release() {
	_ = l.f.Close()
	_ = os.Remove(l.path)
}
