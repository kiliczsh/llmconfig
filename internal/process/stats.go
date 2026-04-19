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
		"wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid),
		"get", "WorkingSetSize,PercentProcessorTime", "/format:csv",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("stats: wmic: %w", err)
	}

	stats := &ProcessStats{PID: pid}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) >= 3 {
			if mem, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
				stats.MemoryMB = mem / 1024 / 1024
			}
		}
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
