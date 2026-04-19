package state

import (
	"fmt"
	"os"
	"time"
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
			return &fileLock{path: lockPath, f: f}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("state lock: %w", err)
		}

		// Check for stale lock
		if info, statErr := os.Stat(lockPath); statErr == nil {
			if time.Since(info.ModTime()) > staleAge {
				_ = os.Remove(lockPath)
				continue
			}
		}

		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("state lock: could not acquire after %d retries", maxRetries)
}

func (l *fileLock) release() {
	_ = l.f.Close()
	_ = os.Remove(l.path)
}
