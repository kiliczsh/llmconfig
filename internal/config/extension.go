package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config files use the .llmc extension going forward. .yaml is still
// accepted on read for backward compatibility with installs that predate
// the extension change; new files are always written with .llmc.
const (
	ExtPrimary = ".llmc"
	ExtLegacy  = ".yaml"
)

// allExts lists the extensions ListConfigs / Find inspect, in priority
// order — .llmc wins when a name happens to exist with both extensions.
var allExts = []string{ExtPrimary, ExtLegacy}

// ConfigPath returns the canonical filesystem path for a config name in
// dir. Always uses ExtPrimary — call this when writing a new config.
func ConfigPath(dir, name string) string {
	return filepath.Join(dir, name+ExtPrimary)
}

// FindConfigInDir returns the path to an existing config file in dir,
// preferring ExtPrimary over ExtLegacy. Returns os.ErrNotExist (wrapped)
// when neither file exists.
func FindConfigInDir(dir, name string) (string, error) {
	for _, ext := range allExts {
		p := filepath.Join(dir, name+ext)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("config: %q not found in %s", name, dir)
}

// IsConfigFile reports whether a basename should be treated as a config
// file. Used by directory walkers that need to filter siblings.
func IsConfigFile(basename string) bool {
	for _, ext := range allExts {
		if strings.HasSuffix(basename, ext) {
			return true
		}
	}
	return false
}

// TrimConfigExt strips either supported extension from a filename. When
// the input has neither, it's returned unchanged.
func TrimConfigExt(basename string) string {
	for _, ext := range allExts {
		if strings.HasSuffix(basename, ext) {
			return strings.TrimSuffix(basename, ext)
		}
	}
	return basename
}

// ListConfigNames returns the deduplicated names (without extension) of
// every config in dir, sorted alphabetically. A name that exists with
// both extensions is reported once (the ExtPrimary file wins on
// disambiguation order, but the dedup is by name so it doesn't matter
// for the caller).
func ListConfigNames(dir string) ([]string, error) {
	seen := map[string]struct{}{}
	for _, ext := range allExts {
		matches, err := filepath.Glob(filepath.Join(dir, "*"+ext))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			name := strings.TrimSuffix(filepath.Base(m), ext)
			seen[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}

// ListConfigPaths returns absolute paths to every config in dir, with
// ExtPrimary entries listed before ExtLegacy. Useful for tooling that
// needs to read every config (archive bundling, validation sweeps).
// When a name appears under both extensions, only the ExtPrimary path
// is returned — the legacy file is hidden so the caller never sees a
// duplicate config name.
func ListConfigPaths(dir string) ([]string, error) {
	seen := map[string]string{} // name → chosen path
	for _, ext := range allExts {
		matches, err := filepath.Glob(filepath.Join(dir, "*"+ext))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			name := strings.TrimSuffix(filepath.Base(m), ext)
			if _, taken := seen[name]; taken {
				continue
			}
			seen[name] = m
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = seen[n]
	}
	return out, nil
}
