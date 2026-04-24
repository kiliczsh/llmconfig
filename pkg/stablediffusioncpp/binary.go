package stablediffusioncpp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/dirs"
)

// BinDir returns the directory where llmconfig manages stable-diffusion.cpp binaries.
func BinDir() string {
	return filepath.Join(dirs.BaseDir(), "bin", "sd")
}

// FindServer returns the path to the sd-server binary, preferring the
// managed bin dir. It deliberately does NOT fall back to sd-cli: `up`
// launches this path with server-only flags, and starting sd-cli with
// those flags just fails confusingly.
func FindServer() (string, error) {
	candidates := []string{"sd-server"}
	if runtime.GOOS == "windows" {
		candidates = []string{"sd-server.exe"}
	}

	for _, name := range candidates {
		p := filepath.Join(BinDir(), name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("sd-server not found — run: llmconfig install sd")
}

// FindBinary returns the path to the sd-cli binary, preferring the managed bin dir.
func FindBinary() (string, error) {
	candidates := []string{"sd-cli", "sd-server", "sd"}
	if runtime.GOOS == "windows" {
		candidates = []string{"sd-cli.exe", "sd-server.exe", "sd.exe"}
	}

	// 1. Managed bin dir
	for _, name := range candidates {
		p := filepath.Join(BinDir(), name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. PATH
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("sd-cli not found — run: llmconfig install sd")
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
