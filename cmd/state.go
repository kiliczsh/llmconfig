package cmd

import "github.com/spf13/cobra"

func newStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "State file maintenance",
	}
	cmd.AddCommand(newStatePruneCmd())
	return cmd
}

func newStatePruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Mark dead-PID running entries as stopped",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			removed, err := appCtx.StateStore.PruneStale()
			if err != nil {
				return err
			}

			for _, name := range removed {
				p.Info("marked %s stopped", name)
			}
			p.Success("pruned %d entries", len(removed))
			return nil
		},
	}
}
