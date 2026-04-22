//go:build !windows

package process

import "syscall"

func pidAlive(pid int) bool {
	// Signal 0 doesn't deliver a signal but performs error checking.
	// ESRCH → process gone; EPERM → exists but we can't signal it.
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return err == syscall.EPERM
}
