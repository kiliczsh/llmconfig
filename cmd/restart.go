package cmd

import (
	"context"
	"time"

	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/hardware"
	"github.com/kiliczsh/llmconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	var flagAll bool
	var flagTimeout time.Duration

	cmd := &cobra.Command{
		Use:               "restart [name]",
		Short:             "Stop and start a model",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeRunningModels,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

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

			r := runner.New()
			for _, name := range targets {
				ms, ok := sf.Models[name]
				if !ok {
					p.Warn("model %q not found in state", name)
					continue
				}

				// Stop
				if ms.Status == "running" {
					p.Info("stopping %s...", name)
					if err := r.Stop(cmd.Context(), ms, flagTimeout); err != nil {
						p.Error("stop %q: %v", name, err)
						continue
					}
					ms.Status = "stopped"
					if err := appCtx.StateStore.Put(ms); err != nil {
						p.Warn("could not persist stopped state for %q: %v", name, err)
					}
				}

				// Reload config and start
				cfg, err := config.Load(name, appCtx.ConfigDir)
				if err != nil {
					p.Error("reload config %q: %v", name, err)
					continue
				}
				config.ApplyDefaults(cfg)

				hw := hardware.Detect()
				binaryPath, err := resolveBackendBinary(cfg.Backend)
				if err != nil {
					p.Error("resolve binary %q: %v", name, err)
					continue
				}
				rc, err := config.Resolve(cfg, hw, binaryPath)
				if err != nil {
					p.Error("resolve %q: %v", name, err)
					continue
				}

				p.Info("starting %s...", name)
				newMS, err := r.Start(context.Background(), rc)
				if err != nil {
					p.Error("start %q: %v", name, err)
					continue
				}

				if err := appCtx.StateStore.Put(newMS); err != nil {
					p.Warn("could not save state for %q: %v", name, err)
				}

				if err := runner.WaitHealthy(cmd.Context(), cfg.Server.Host, cfg.Server.Port, cfg.Backend); err != nil {
					p.Error("%s health check failed: %v", name, err)
					// Mirror `up`: don't leave the entry marked "running" when
					// startup failed. `ps` would otherwise show a ghost.
					newMS.Status = "error"
					if putErr := appCtx.StateStore.Put(newMS); putErr != nil {
						p.Warn("could not save error state for %q: %v", name, putErr)
					}
					continue
				}

				p.Success("restarted %s on port %d", name, cfg.Server.Port)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "restart all running models")
	cmd.Flags().DurationVar(&flagTimeout, "timeout", 10*time.Second, "how long to wait before force kill (e.g. 30s, 1m)")
	return cmd
}
