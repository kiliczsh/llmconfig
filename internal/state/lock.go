package state

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kiliczsh/llamaconfig/internal/process"
)

// ErrLockHeld is returned by tryAcquireLock when the lock file exists and
// its holder process is still alive. Callers typically present this as a
// "already in progress, try again later" message to the user.
var ErrLockHeld = errors.New("lock is held by another process")

const staleLockAge = 30 * time.Second

type fileLock struct {
	path string
	f    *os.File
}

// acquireLock blocks (up to ~1s across 20 retries) until the lock is free.
// Used for short state-file mutations where we expect the lock to become
// available quickly.
func acquireLock(lockPath string) (*fileLock, error) {
	const maxRetries = 20
	const retryDelay = 50 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		fl, err := createLockFile(lockPath)
		if err == nil {
			return fl, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("state lock: %w", err)
		}

		if isStaleLock(lockPath, staleLockAge) {
			_ = os.Remove(lockPath)
			continue
		}

		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("state lock: could not acquire after %d retries", maxRetries)
}

// tryAcquireLock makes one attempt (plus one stale-recovery attempt) to
// acquire the lock and returns ErrLockHeld immediately otherwise. Suitable
// for long-lived locks (e.g. per-model during `up`) where we'd rather fail
// fast than wait for an in-progress download to finish.
func tryAcquireLock(lockPath string) (*fileLock, error) {
	fl, err := createLockFile(lockPath)
	if err == nil {
		return fl, nil
	}
	if !os.IsExist(err) {
		return nil, fmt.Errorf("state lock: %w", err)
	}

	if isStaleLock(lockPath, staleLockAge) {
		_ = os.Remove(lockPath)
		if fl, err := createLockFile(lockPath); err == nil {
			return fl, nil
		}
	}
	return nil, ErrLockHeld
}

func createLockFile(lockPath string) (*fileLock, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(f, "%d", os.Getpid())
	return &fileLock{path: lockPath, f: f}, nil
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
