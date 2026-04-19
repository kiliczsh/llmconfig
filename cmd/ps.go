package cmd

import (
	"fmt"
	"sort"

	"github.com/kiliczsh/llamaconfig/internal/output"
	"github.com/kiliczsh/llamaconfig/internal/runner"
	"github.com/kiliczsh/llamaconfig/internal/state"
	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	var flagAll bool
	var flagFormat string

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List running models",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			r := runner.New()

			sf, err := appCtx.StateStore.Load()
			if err != nil {
				return err
			}

			// Sort by name for stable output
			names := make([]string, 0, len(sf.Models))
			for name := range sf.Models {
				names = append(names, name)
			}
			sort.Strings(names)

			// Update liveness
			for _, name := range names {
				ms := sf.Models[name]
				if ms.Status == "running" && !r.IsAlive(ms) {
					ms.Status = "stopped"
				}
			}

			// Filter
			var shown []*state.ModelState
			for _, name := range names {
				ms := sf.Models[name]
				if flagAll || ms.Status == "running" {
					shown = append(shown, ms)
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

			switch flagFormat {
			case "json":
				return p.PrintJSON(shown)
			case "yaml":
				return p.PrintYAML(shown)
			default:
				return renderPsTable(p, shown)
			}
		},
	}

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "include stopped models")
	cmd.Flags().StringVar(&flagFormat, "format", "table", "output format: table | json | yaml")
	return cmd
}

func renderPsTable(p *output.Printer, models []*state.ModelState) error {
	headers := []string{"NAME", "STATUS", "PORT", "PROFILE", "UPTIME", "PID"}
	rows := make([][]string, len(models))
	for i, ms := range models {
		rows[i] = []string{
			ms.Name,
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
