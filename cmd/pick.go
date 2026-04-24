package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/kiliczsh/llmconfig/internal/dirs"
	"github.com/kiliczsh/llmconfig/internal/state"
	"github.com/spf13/cobra"
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
	return selected, nil
}

// completeConfigNames is a cobra ValidArgsFunction that completes model config
// names from the default config directory.
func completeConfigNames(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	entries, err := os.ReadDir(dirs.ConfigDir())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeRunningModels is a cobra ValidArgsFunction that completes names of
// currently running models from the state store.
func completeRunningModels(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	sf, err := state.NewStore().Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for name, ms := range sf.Models {
		if ms.Status == "running" {
			names = append(names, name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
