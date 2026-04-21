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
	name := "sd"
	if runtime.GOOS == "windows" {
		name = "sd.exe"
	}

	// 1. Managed bin dir
	managed := filepath.Join(BinDir(), name)
	if _, err := os.Stat(managed); err == nil {
		return managed, nil
	}

	// 2. PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
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
