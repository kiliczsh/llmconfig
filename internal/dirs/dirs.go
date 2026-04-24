package dirs

import (
	"os"
	"path/filepath"
)

func ConfigDir() string {
	return filepath.Join(BaseDir(), "configs")
}

func CacheDir() string {
	return filepath.Join(BaseDir(), "cache")
}

func LogDir() string {
	return filepath.Join(BaseDir(), "logs")
}

func BenchDir() string {
	return filepath.Join(BaseDir(), "bench")
}

func StateFile() string {
	return filepath.Join(BaseDir(), "state.json")
}

func StateLockFile() string {
	return filepath.Join(BaseDir(), "state.json.lock")
}

// ModelLockDir is the directory holding per-model lock files for serialising
// concurrent operations (e.g. `llamaconfig up X`) against the same model.
func ModelLockDir() string {
	return filepath.Join(BaseDir(), "locks")
}

// ModelLockFile is the path of the lock file for a single model.
func ModelLockFile(name string) string {
	return filepath.Join(ModelLockDir(), name+".lock")
}

func EnsureAll() error {
	for _, d := range []string{ConfigDir(), CacheDir(), LogDir(), BenchDir()} {
		if err := EnsureDir(d); err != nil {
			return err
		}
	}
	return nil
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// BaseDir returns the llamaconfig root directory (LLAMACONFIG_CONFIG_DIR or
// the user's home dir joined with ".llamaconfig"). It is platform-independent:
// os.UserHomeDir resolves to %USERPROFILE% on Windows and $HOME elsewhere.
func BaseDir() string {
	if v := os.Getenv("LLAMACONFIG_CONFIG_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".llamaconfig")
}

// ExpandHome resolves leading "~" to the user's home directory. Handles
// bare "~", "~/<rest>", and "~\<rest>" (the last form is what Windows
// users typically type when editing YAML by hand).
func ExpandHome(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if len(path) >= 2 && (path[:2] == "~/" || path[:2] == `~\`) {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
