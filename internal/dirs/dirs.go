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

func ExpandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
