//go:build windows

package runner

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func isProcessAlive(proc *os.Process) bool {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(proc.Pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)

	var code uint32
	if err := syscall.GetExitCodeProcess(handle, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

func stopWindows(pid int, timeoutSec int) error {
	cmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid))
	if err := cmd.Run(); err != nil {
		// Try force kill
		_ = exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid), "/F").Run()
	}
	time.Sleep(time.Duration(timeoutSec) * time.Second)
	return nil
}
