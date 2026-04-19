package hardware

import (
	"os/exec"
	"runtime"
	"strings"
)

// Detect probes the current system and returns hardware information.
func Detect() *DetectionResult {
	r := &DetectionResult{
		Class:    ClassCPU,
		CPUCores: runtime.NumCPU(),
		IsARM:    runtime.GOARCH == "arm64",
	}

	switch runtime.GOOS {
	case "darwin":
		detectDarwin(r)
	case "linux":
		detectLinux(r)
	case "windows":
		detectWindows(r)
	}

	return r
}

func detectDarwin(r *DetectionResult) {
	if runtime.GOARCH == "arm64" {
		r.Class = ClassAppleSilicon
		r.GPUName = "Apple Silicon"
		// Unified memory — report system RAM as VRAM approximation
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			var bytes uint64
			_, _ = strings.NewReader(strings.TrimSpace(string(out))), &bytes
		}
		return
	}
	// Intel Mac — check for discrete GPU via system_profiler
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return
	}
	s := string(out)
	if strings.Contains(strings.ToLower(s), "nvidia") {
		r.Class = ClassNVIDIA
		r.GPUName = extractGPUName(s, "nvidia")
	} else if strings.Contains(strings.ToLower(s), "amd") || strings.Contains(strings.ToLower(s), "radeon") {
		r.Class = ClassAMD
		r.GPUName = extractGPUName(s, "amd")
	}
}

func detectLinux(r *DetectionResult) {
	// Check NVIDIA first
	out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader").Output()
	if err == nil && len(out) > 0 {
		r.Class = ClassNVIDIA
		parts := strings.SplitN(strings.TrimSpace(string(out)), ",", 2)
		if len(parts) >= 1 {
			r.GPUName = strings.TrimSpace(parts[0])
		}
		return
	}

	// Check AMD via sysfs
	out, err = exec.Command("sh", "-c", "cat /sys/class/drm/card0/device/vendor 2>/dev/null").Output()
	if err == nil {
		vendor := strings.TrimSpace(string(out))
		if vendor == "0x1002" {
			r.Class = ClassAMD
			r.GPUName = "AMD GPU"
			return
		}
		if vendor == "0x8086" {
			r.Class = ClassIntelGPU
			r.GPUName = "Intel GPU"
			return
		}
	}
}

func detectWindows(r *DetectionResult) {
	out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader").Output()
	if err == nil && len(out) > 0 {
		r.Class = ClassNVIDIA
		parts := strings.SplitN(strings.TrimSpace(string(out)), ",", 2)
		if len(parts) >= 1 {
			r.GPUName = strings.TrimSpace(parts[0])
		}
		return
	}

	// Check via wmic
	out, err = exec.Command("wmic", "path", "win32_VideoController", "get", "name").Output()
	if err == nil {
		s := strings.ToLower(string(out))
		if strings.Contains(s, "nvidia") {
			r.Class = ClassNVIDIA
		} else if strings.Contains(s, "amd") || strings.Contains(s, "radeon") {
			r.Class = ClassAMD
		} else if strings.Contains(s, "intel") {
			r.Class = ClassIntelGPU
		}
	}
}

func extractGPUName(s, vendor string) string {
	for _, line := range strings.Split(s, "\n") {
		l := strings.ToLower(line)
		if strings.Contains(l, vendor) || strings.Contains(l, "chipset model") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return strings.ToUpper(vendor) + " GPU"
}
