package whispercpp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/dirs"
)

// BinDir returns the directory where llmconfig manages whisper.cpp binaries.
func BinDir() string {
	return filepath.Join(dirs.BaseDir(), "bin", "whisper")
}

// FindServer returns the path to the whisper-server binary, preferring
// the managed bin dir. It deliberately does NOT fall back to whisper-cli:
// `up` launches this path with server-only flags, and starting the CLI
// with those flags just fails confusingly.
func FindServer() (string, error) {
	candidates := []string{"whisper-server"}
	if runtime.GOOS == "windows" {
		candidates = []string{"whisper-server.exe"}
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
	return "", fmt.Errorf("whisper-server not found — run: llmconfig install whisper")
}

// FindBinary returns the path to the whisper-cli binary, preferring the managed bin dir.
func FindBinary() (string, error) {
	candidates := []string{"whisper-cli", "whisper-server"}
	if runtime.GOOS == "windows" {
		candidates = []string{"whisper-cli.exe", "whisper-server.exe"}
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

	return "", fmt.Errorf("whisper-cli not found — run: llmconfig install whisper")
}

// Version runs whisper-cli and tries to extract the version string.
// whisper-cli does not support --version; we parse whatever output is available.
func Version(binPath string) (string, error) {
	out, _ := exec.Command(binPath, "--version").CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") || strings.HasPrefix(line, "whisper") {
			return line, nil
		}
	}
	return "version unknown", nil
}
