package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load finds and loads a named config. Searches:
//  1. ./llamaconfig/<name>.yaml (current dir)
//  2. configDir/<name>.yaml
//  3. <name> treated as a direct file path
func Load(name, configDir string) (*Config, error) {
	path, err := findConfig(name, configDir)
	if err != nil {
		return nil, err
	}
	return LoadFile(path)
}

// LoadFile loads a config from an explicit file path.
func LoadFile(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	// Expand ${ENV_VAR} references before unmarshalling.
	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	cfg.FilePath = path
	return &cfg, nil
}

func findConfig(name, configDir string) (string, error) {
	candidates := []string{
		filepath.Join("llamaconfig", name+".yaml"),
		filepath.Join(configDir, name+".yaml"),
		name,
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, err := filepath.Abs(p)
			if err != nil {
				return p, nil
			}
			return abs, nil
		}
	}

	return "", fmt.Errorf("config: %q not found (searched %v)", name, candidates)
}
