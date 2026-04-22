package process

// PidAlive reports whether a process with the given PID is still running.
// Returns false on any error (including permission errors on platforms
// where we can't tell).
func PidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return pidAlive(pid)
}
