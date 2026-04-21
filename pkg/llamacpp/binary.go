package llamacpp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llamaconfig/internal/dirs"
)

// BinDir returns the directory where llamaconfig manages llama.cpp binaries.
func BinDir() string {
	return filepath.Join(dirs.ExpandHome("~/.llamaconfig"), "bin", "llama")
}

// FindServer returns the path to llama-server, preferring the managed bin dir.
func FindServer() (string, error) {
	name := "llama-server"
	if runtime.GOOS == "windows" {
		name = "llama-server.exe"
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

	return "", fmt.Errorf("llama-server not found — run: llamaconfig install llama")
}

// Version runs llama-server --version and returns the version string.
func Version(binPath string) (string, error) {
	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("llama-server --version: %w", err)
	}
	// Extract the "version: XXXX (hash)" line from potentially noisy output
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") {
			return line, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}
