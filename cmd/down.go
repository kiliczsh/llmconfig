package cmd

import (
	"time"

	"github.com/kiliczsh/llmconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	var flagAll bool
	var flagTimeout time.Duration

	cmd := &cobra.Command{
		Use:               "down [name]",
		Short:             "Stop a running model",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeRunningModels,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			r := runner.New()

			sf, err := appCtx.StateStore.Load()
			if err != nil {
				return err
			}

			var targets []string
			if flagAll {
				for name, ms := range sf.Models {
					if ms.Status == "running" {
						targets = append(targets, name)
					}
				}
			} else {
				var arg string
				if len(args) > 0 {
					arg = args[0]
				}
				name, err := pickRunningModel(arg, sf)
				if err != nil {
					return err
				}
				targets = []string{name}
			}

			if len(targets) == 0 {
				p.Info("no running models to stop")
				return nil
			}

			for _, name := range targets {
				ms, ok := sf.Models[name]
				if !ok {
					p.Warn("model %q not found in state", name)
					continue
				}
				if ms.Status != "running" {
					p.Warn("model %q is not running (status: %s)", name, ms.Status)
					continue
				}

				if err := r.Stop(cmd.Context(), ms, flagTimeout); err != nil {
					p.Error("stop %q: %v", name, err)
					continue
				}

				ms.Status = "stopped"
				if err := appCtx.StateStore.Put(ms); err != nil {
					p.Warn("state update failed for %q: %v", name, err)
				}

				p.Success("stopped %s", name)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "stop all running models")
	cmd.Flags().DurationVar(&flagTimeout, "timeout", 10*time.Second, "how long to wait before force kill (e.g. 30s, 1m)")
	return cmd
}
