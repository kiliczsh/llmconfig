// Package ikllamacpp manages the ik_llama.cpp fork
// (https://github.com/ikawrakow/ik_llama.cpp) as an alternative engine for the
// "llama" backend. The fork ships no prebuilt release assets, so installation
// is either source-based (git clone + cmake) or bring-your-own-binary via a
// local archive.
package ikllamacpp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/dirs"
)

// BinDir returns the directory where llmconfig stores ik_llama.cpp binaries.
// Kept separate from pkg/llamacpp.BinDir() so both engines can coexist.
func BinDir() string {
	return filepath.Join(dirs.BaseDir(), "bin", "ik-llama")
}

// CacheDir returns the directory where the ik_llama.cpp source tree is kept
// between builds. Persisting it makes `git pull` + incremental rebuild fast.
func CacheDir() string {
	return filepath.Join(dirs.BaseDir(), "cache", "ik_llama.cpp")
}

// FindServer returns the path to the ik_llama.cpp llama-server binary,
// preferring the managed bin dir. Falls back to PATH only if the binary's
// parent directory equals BinDir() — we never want to silently pick up the
// upstream llama-server from PATH and call it ik_llama.
func FindServer() (string, error) {
	name := "llama-server"
	if runtime.GOOS == "windows" {
		name = "llama-server.exe"
	}

	managed := filepath.Join(BinDir(), name)
	if _, err := os.Stat(managed); err == nil {
		return managed, nil
	}

	return "", fmt.Errorf("ik_llama.cpp llama-server not found in %s — run: llmconfig install ik-llama", BinDir())
}

// Version runs the binary with --version and returns the first "version:" line.
func Version(binPath string) (string, error) {
	out, err := exec.Command(binPath, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ik_llama-server --version: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") {
			return line, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}
