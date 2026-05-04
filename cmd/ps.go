package cmd

import (
	"fmt"
	"sort"

	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/output"
	"github.com/kiliczsh/llmconfig/internal/runner"
	"github.com/kiliczsh/llmconfig/internal/state"
	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	var flagAll bool

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List running models",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			r := runner.New()

			// Reconcile liveness: flip any dead "running" entries to "stopped"
			// and persist, so subsequent reads see fresh state.
			sf, err := reconcileLiveness(appCtx.StateStore, r, p)
			if err != nil {
				return err
			}

			// Sort by name for stable output
			names := make([]string, 0, len(sf.Models))
			for name := range sf.Models {
				names = append(names, name)
			}
			sort.Strings(names)

			// Filter server-mode models from state
			var shown []*state.ModelState
			for _, name := range names {
				ms := sf.Models[name]
				if flagAll || ms.Status == "running" {
					shown = append(shown, ms)
				}
			}

			// Add interactive-mode models from lock files
			if matches, err := config.ListConfigPaths(appCtx.ConfigDir); err == nil {
				for _, path := range matches {
					cfg, err := config.LoadFile(path)
					if err != nil || cfg.Mode != "interactive" {
						continue
					}
					if _, already := sf.Models[cfg.Name]; already {
						continue
					}
					if appCtx.StateStore.IsInteractiveRunning(cfg.Name) {
						shown = append(shown, &state.ModelState{
							Name:    cfg.Name,
							Backend: cfg.Backend,
							Status:  "running",
						})
					} else if flagAll {
						shown = append(shown, &state.ModelState{
							Name:    cfg.Name,
							Backend: cfg.Backend,
							Status:  "stopped",
						})
					}
				}
			}

			if len(shown) == 0 {
				if flagAll {
					p.Info("no models")
				} else {
					p.Info("no running models (use --all to show stopped models)")
				}
				return nil
			}

			return renderPsTable(p, shown)
		},
	}

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "include stopped models")
	return cmd
}

func renderPsTable(p *output.Printer, models []*state.ModelState) error {
	headers := []string{"NAME", "BACKEND", "STATUS", "PORT", "PROFILE", "UPTIME", "PID"}
	rows := make([][]string, len(models))
	for i, ms := range models {
		backend := ms.Backend
		if backend == "" {
			backend = "llama"
		}
		rows[i] = []string{
			ms.Name,
			backend,
			output.StatusColor(ms.Status),
			fmt.Sprintf("%d", ms.Port),
			ms.ProfileName,
			output.FormatUptime(ms.StartedAt),
			fmt.Sprintf("%d", ms.PID),
		}
	}
	p.Table(headers, rows)
	return nil
}
