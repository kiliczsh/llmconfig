package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/dirs"
	"github.com/kiliczsh/llmconfig/internal/state"
)

type serverRunner struct{}

func (r *serverRunner) Start(ctx context.Context, rc *config.RunConfig) (*state.ModelState, error) {
	if err := dirs.EnsureDir(filepath.Dir(rc.LogFile)); err != nil {
		return nil, fmt.Errorf("runner: create log dir: %w", err)
	}

	logFile, err := os.OpenFile(rc.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("runner: open log file: %w", err)
	}

	args := BuildArgs(rc)
	cmd := exec.CommandContext(ctx, rc.BinaryPath, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("runner: start %s: %w", filepath.Base(rc.BinaryPath), err)
	}

	pid := cmd.Process.Pid
	// Release so Go runtime doesn't wait for the child
	_ = cmd.Process.Release()
	logFile.Close()

	ms := &state.ModelState{
		Name:        rc.Config.Name,
		PID:         pid,
		Port:        rc.Config.Server.Port,
		Host:        rc.Config.Server.Host,
		ConfigPath:  rc.Config.FilePath,
		LogFile:     rc.LogFile,
		StartedAt:   time.Now(),
		ProfileName: rc.ProfileName,
		Status:      "running",
		BinaryPath:  rc.BinaryPath,
		Backend:     rc.Backend,
	}

	return ms, nil
}

func (r *serverRunner) Stop(ctx context.Context, ms *state.ModelState, timeout time.Duration) error {
	proc, err := os.FindProcess(ms.PID)
	if err != nil {
		return nil // already gone
	}

	if runtime.GOOS == "windows" {
		return stopWindows(ms.PID, timeout)
	}

	// SIGTERM — graceful shutdown; llama/sd/whisper servers all handle it.
	// SIGINT (os.Interrupt) was previously used but is Ctrl-C semantics and
	// not guaranteed to reach the process in the same way.
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		return nil
	}

	// Wait up to timeout, then SIGKILL
	deadline := time.After(timeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = proc.Kill()
			return ctx.Err()
		case <-deadline:
			_ = proc.Kill()
			return nil
		case <-ticker.C:
			if !r.IsAlive(ms) {
				return nil
			}
		}
	}
}

func (r *serverRunner) IsAlive(ms *state.ModelState) bool {
	proc, err := os.FindProcess(ms.PID)
	if err != nil {
		return false
	}
	return isProcessAlive(proc)
}
