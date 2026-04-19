package cmd

import (
	"fmt"

	"github.com/kiliczsh/llamaconfig/internal/output"
	"github.com/kiliczsh/llamaconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Show detailed status of a model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			ms, err := appCtx.StateStore.Get(name)
			if err != nil {
				return err
			}
			if ms == nil {
				return fmt.Errorf("model %q not found in state", name)
			}

			r := runner.New()
			if ms.Status == "running" && !r.IsAlive(ms) {
				ms.Status = "stopped"
				_ = appCtx.StateStore.Put(ms)
			}

			if appCtx.JSONOutput {
				return p.PrintJSON(ms)
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
