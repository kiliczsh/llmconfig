package stablediffusioncpp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llamaconfig/internal/dirs"
)

// BinDir returns the directory where llamaconfig manages stable-diffusion.cpp binaries.
func BinDir() string {
	return filepath.Join(dirs.ExpandHome("~/.llamaconfig"), "bin")
}

// FindBinary returns the path to the sd binary, preferring the managed bin dir.
func FindBinary() (string, error) {
	// 1. Well-known names in managed bin dir
	candidates := []string{"sd", "stable-diffusion"}
	if runtime.GOOS == "windows" {
		candidates = []string{"sd.exe", "stable-diffusion.exe"}
	}
	for _, name := range candidates {
		p := filepath.Join(BinDir(), name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. Scan managed bin dir for any exe starting with "sd"
	if entries, err := os.ReadDir(BinDir()); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lower := strings.ToLower(e.Name())
			if strings.HasPrefix(lower, "sd") && (runtime.GOOS == "windows") == strings.HasSuffix(lower, ".exe") {
				return filepath.Join(BinDir(), e.Name()), nil
			}
		}
	}

	// 3. PATH
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("sd not found — run: llamaconfig sd --install")
}

// Version runs sd --version and returns the version string.
func Version(binPath string) (string, error) {
	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("sd --version: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") || strings.HasPrefix(line, "stable-diffusion") {
			return line, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}
