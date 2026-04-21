package whispercpp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llamaconfig/internal/dirs"
)

// BinDir returns the directory where llamaconfig manages whisper.cpp binaries.
func BinDir() string {
	return filepath.Join(dirs.ExpandHome("~/.llamaconfig"), "bin")
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

	return "", fmt.Errorf("whisper-cli not found — run: llamaconfig whisper --install")
}

// Version runs whisper-cli --version and returns the version string.
func Version(binPath string) (string, error) {
	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper-cli --version: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") || strings.HasPrefix(line, "whisper") {
			return line, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}
