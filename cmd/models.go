package cmd

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/output"
	"github.com/kiliczsh/llmconfig/internal/runner"
	"github.com/kiliczsh/llmconfig/internal/state"
	"github.com/spf13/cobra"
)

type modelEntry struct {
	Name       string
	Backend    string
	Mode       string
	Source     string
	File       string
	CachedSize string
	Status     string
	Port       string
}

func newModelsCmd() *cobra.Command {
	var flagRunning bool
	var flagCached bool

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List all models (running + stopped + cached)",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			r := runner.New()

			sf, err := reconcileLiveness(appCtx.StateStore, r, p)
			if err != nil {
				return err
			}

			entries, err := buildModelEntries(appCtx.ConfigDir, appCtx.ModelsDir, sf, r, appCtx.StateStore)
			if err != nil {
				return err
			}

			var shown []modelEntry
			for _, e := range entries {
				if flagRunning && e.Status != "running" {
					continue
				}
				if flagCached && e.CachedSize == "-" {
					continue
				}
				shown = append(shown, e)
			}

			if len(shown) == 0 {
				p.Info("no models found")
				return nil
			}

			return renderModelsTable(p, shown)
		},
	}

	cmd.Flags().BoolVar(&flagRunning, "running", false, "show only running models")
	cmd.Flags().BoolVar(&flagCached, "cached", false, "show only downloaded models")
	return cmd
}

func buildModelEntries(configDir, modelsDir string, sf *state.StateFile, r runner.Runner, ss *state.Store) ([]modelEntry, error) {
	pattern := filepath.Join(configDir, "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)

	var entries []modelEntry
	for _, path := range matches {
		cfg, err := config.LoadFile(path)
		if err != nil {
			continue
		}
		config.ApplyDefaults(cfg)

		e := modelEntry{
			Name:    cfg.Name,
			Backend: cfg.Backend,
			Mode:    cfg.Mode,
			Source:  cfg.Model.Source,
			File:    cfg.Model.File,
			Status:  "stopped",
			Port:    "-",
		}
		if cfg.Model.Source == "local" {
			e.File = filepath.Base(cfg.Model.Path)
		}

		// Check if downloaded
		cacheFile := filepath.Join(modelsDir, cfg.Model.File)
		if cfg.Model.Source == "local" {
			cacheFile = cfg.Model.Path
		}
		if info, err := os.Stat(cacheFile); err == nil {
			e.CachedSize = humanize.Bytes(uint64(info.Size()))
		} else {
			e.CachedSize = "-"
		}

		// Overlay running state
		if cfg.Mode == "interactive" {
			if ss.IsInteractiveRunning(cfg.Name) {
				e.Status = "running"
			}
		} else if ms, ok := sf.Models[cfg.Name]; ok {
			if ms.Status == "running" && r.IsAlive(ms) {
				e.Status = "running"
				e.Port = output.Bold(formatPort(ms.Port))
			}
		}

		entries = append(entries, e)
	}
	return entries, nil
}

func renderModelsTable(p *output.Printer, models []modelEntry) error {
	headers := []string{"NAME", "BACKEND", "MODE", "SOURCE", "CACHED", "STATUS", "PORT"}
	rows := make([][]string, len(models))
	for i, e := range models {
		rows[i] = []string{
			e.Name,
			e.Backend,
			e.Mode,
			e.Source,
			e.CachedSize,
			output.StatusColor(e.Status),
			e.Port,
		}
	}
	p.Table(headers, rows)
	return nil
}

func formatPort(port int) string {
	if port == 0 {
		return "-"
	}
	return output.Bold(itoa(port))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-(n-3):]
}
