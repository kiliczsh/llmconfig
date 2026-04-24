package cmd

import (
	"fmt"

	"github.com/kiliczsh/llmconfig/internal/output"
	"github.com/kiliczsh/llmconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "status [name]",
		Short:             "Show detailed status of a model",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeRunningModels,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			sf, err := appCtx.StateStore.Load()
			if err != nil {
				return err
			}
			var arg string
			if len(args) > 0 {
				arg = args[0]
			}
			name, err := pickRunningModel(arg, sf)
			if err != nil {
				return err
			}
			ms, ok := sf.Models[name]
			if !ok || ms == nil {
				return fmt.Errorf("model %q not found in state", name)
			}

			r := runner.New()
			if ms.Status == "running" && !r.IsAlive(ms) {
				ms.Status = "stopped"
				if putErr := appCtx.StateStore.Put(ms); putErr != nil {
					p.Warn("could not persist stopped state for %q: %v", name, putErr)
				}
			}

			rows := [][]string{
				{"Name", ms.Name},
				{"Status", output.StatusColor(ms.Status)},
				{"PID", fmt.Sprintf("%d", ms.PID)},
				{"Port", fmt.Sprintf("%d", ms.Port)},
				{"Host", ms.Host},
				{"Profile", ms.ProfileName},
				{"Uptime", output.FormatUptime(ms.StartedAt)},
				{"Started", output.FormatTime(ms.StartedAt)},
				{"Config", ms.ConfigPath},
				{"Log", ms.LogFile},
				{"Binary", ms.BinaryPath},
			}

			for _, row := range rows {
				fmt.Printf("  %-10s  %s\n", output.Bold(row[0])+":", row[1])
			}
			return nil
		},
	}
}
