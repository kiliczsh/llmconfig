package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load finds and loads a named config. Searches in this order, trying
// both the canonical .llmc and the legacy .yaml extension at each step:
//  1. ./llmconfig/<name>.{llmc,yaml} (current dir)
//  2. configDir/<name>.{llmc,yaml}
//  3. <name> treated as a direct file path
//
// .llmc wins when both extensions are present in the same directory.
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

	// KnownFields(true) surfaces typos (e.g. `parral: 2` instead of
	// `parallel: 2`) at load time instead of letting them silently do
	// nothing.
	dec := yaml.NewDecoder(bytes.NewReader([]byte(expanded)))
	dec.KnownFields(true)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	cfg.FilePath = path
	return &cfg, nil
}

func findConfig(name, configDir string) (string, error) {
	// Build the search list: each directory tried with both extensions,
	// then the bare name as a literal path. The order matters: .llmc
	// always beats .yaml within a single directory, but later
	// directories never override earlier ones.
	var candidates []string
	for _, dir := range []string{"llmconfig", configDir} {
		for _, ext := range allExts {
			candidates = append(candidates, filepath.Join(dir, name+ext))
		}
	}
	candidates = append(candidates, name)

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
