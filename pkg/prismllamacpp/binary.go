// Package prismllamacpp manages Prism ML's llama.cpp fork as a separate
// llama-compatible inference backend.
package prismllamacpp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/dirs"
)

const RepoURL = "https://github.com/PrismML-Eng/llama.cpp"

// BinDir is intentionally separate from stock llama.cpp so installing or
// updating either engine cannot overwrite the other one's binaries or DLLs.
func BinDir() string {
	return filepath.Join(dirs.BaseDir(), "bin", "prism-llama")
}

func FindServer() (string, error) {
	name := "llama-server"
	if runtime.GOOS == "windows" {
		name = "llama-server.exe"
	}
	path := filepath.Join(BinDir(), name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("Prism llama-server not found in %s — install a Prism ML release there", BinDir())
}

func Version(binPath string) (string, error) {
	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Prism llama-server --version: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") {
			return line, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}
