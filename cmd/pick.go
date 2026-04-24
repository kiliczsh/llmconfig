package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/kiliczsh/llmconfig/internal/state"
)

// pickRunningModel returns a model name to operate on.
// If name is non-empty it is returned as-is.
// If there is exactly one running model it is returned automatically.
// Otherwise an interactive selector is shown.
func pickRunningModel(name string, sf *state.StateFile) (string, error) {
	if name != "" {
		return name, nil
	}

	var running []string
	for n, ms := range sf.Models {
		if ms.Status == "running" {
			running = append(running, n)
		}
	}

	switch len(running) {
	case 0:
		return "", fmt.Errorf("no running models")
	case 1:
		return running[0], nil
	}

	var selected string
	opts := make([]huh.Option[string], len(running))
	for i, n := range running {
		ms := sf.Models[n]
		opts[i] = huh.NewOption(fmt.Sprintf("%s  (port %d, PID %d)", n, ms.Port, ms.PID), n)
	}

	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a running model").
			Options(opts...).
			Value(&selected),
	)).Run()
	if err != nil {
		return "", err
	}
	return selected, nil
}

// pickConfig returns a config name to operate on.
// If name is non-empty it is returned as-is.
// If there is exactly one config it is returned automatically.
// Otherwise an interactive selector is shown.
func pickConfig(name, configDir string) (string, error) {
	if name != "" {
		return name, nil
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return "", fmt.Errorf("cannot read config dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}

	switch len(names) {
	case 0:
		return "", fmt.Errorf("no configs found in %s", configDir)
	case 1:
		return names[0], nil
	}

	var selected string
	opts := make([]huh.Option[string], len(names))
	for i, n := range names {
		opts[i] = huh.NewOption(n, n)
	}

	err = huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a model config").
			Options(opts...).
			Value(&selected),
	)).Run()
	if err != nil {
		return "", err
	}

	_ = filepath.Join // keep import
	return selected, nil
}
