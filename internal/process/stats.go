package process

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type ProcessStats struct {
	PID        int
	CPUPercent float64
	MemoryMB   float64
	Status     string
}

// GetStats returns CPU and memory usage for a given PID.
func GetStats(pid int) (*ProcessStats, error) {
	switch runtime.GOOS {
	case "windows":
		return getStatsWindows(pid)
	default:
		return getStatsUnix(pid)
	}
}

func getStatsWindows(pid int) (*ProcessStats, error) {
	out, err := exec.Command(
		"powershell",
		"-NoProfile",
		"-Command",
		fmt.Sprintf(`$p=Get-CimInstance Win32_Process -Filter 'ProcessId=%d'; $c=Get-CimInstance Win32_PerfFormattedData_PerfProc_Process -Filter 'IDProcess=%d'; Write-Output (""+($c.PercentProcessorTime)+','+($p.WorkingSetSize))`, pid, pid),
	).Output()
	if err != nil {
		return nil, fmt.Errorf("stats: powershell: %w", err)
	}

	// Parse failures (empty output, missing process) fall through to zero
	// stats with no error — same contract as the Unix branch when ps returns
	// fewer columns than expected.
	stats := &ProcessStats{PID: pid}
	fields := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(fields) != 2 {
		return stats, nil
	}
	if cpu, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64); err == nil {
		stats.CPUPercent = cpu
	}
	if memBytes, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
		stats.MemoryMB = memBytes / 1024 / 1024
	}
	return stats, nil
}

func getStatsUnix(pid int) (*ProcessStats, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid,%cpu,rss").Output()
	if err != nil {
		return nil, fmt.Errorf("stats: ps: %w", err)
	}

	stats := &ProcessStats{PID: pid}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return stats, nil
	}
	fields := strings.Fields(lines[1])
	if len(fields) >= 3 {
		if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
			stats.CPUPercent = cpu
		}
		if rss, err := strconv.ParseFloat(fields[2], 64); err == nil {
			stats.MemoryMB = rss / 1024
		}
	}
	return stats, nil
}
