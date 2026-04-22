package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var flagKeepFile bool
	var flagForce bool

	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a model config and optionally its cached file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			r := runner.New()

			configPath := filepath.Join(appCtx.ConfigDir, name+".yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return fmt.Errorf("model %q not found", name)
			}

			cfg, err := config.LoadFile(configPath)
			if err != nil {
				return err
			}
			config.ApplyDefaults(cfg)

			// Check not running
			ms, _ := appCtx.StateStore.Get(name)
			if ms != nil && ms.Status == "running" && r.IsAlive(ms) {
				return fmt.Errorf("model %q is running — stop it first with: llamaconfig down %s", name, name)
			}

			// Determine cached file path
			var cachedFile string
			if !flagKeepFile && cfg.Model.Source != "local" && cfg.Model.File != "" {
				cachedFile = filepath.Join(appCtx.CacheDir, cfg.Model.File)
				if _, err := os.Stat(cachedFile); os.IsNotExist(err) {
					cachedFile = "" // not cached
				}
			}

			// Confirmation prompt
			if !flagForce {
				msg := fmt.Sprintf("Remove model %q?", name)
				if cachedFile != "" {
					msg = fmt.Sprintf("Remove model %q and its cached GGUF file?", name)
				}
				var confirm bool
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title(msg).
							Value(&confirm),
					),
				)
				if err := form.Run(); err != nil {
					return err
				}
				if !confirm {
					p.Info("cancelled")
					return nil
				}
			}

			// Remove config
			if err := os.Remove(configPath); err != nil {
				return fmt.Errorf("rm: remove config: %w", err)
			}
			p.Success("removed config: %s", configPath)

			// Remove cached file
			if cachedFile != "" {
				if err := os.Remove(cachedFile); err != nil {
					p.Warn("could not remove cached file %s: %v", cachedFile, err)
				} else {
					p.Success("removed cached file: %s", cachedFile)
				}
			}

			// Remove from state
			_ = appCtx.StateStore.Remove(name)

			return nil
		},
	}

	cmd.Flags().BoolVar(&flagKeepFile, "keep-file", false, "remove config only, keep the GGUF file in cache")
	cmd.Flags().BoolVarP(&flagForce, "yes", "y", false, "skip confirmation prompt")
	cmd.Flags().BoolVarP(&flagForce, "force", "f", false, "alias for --yes")
	return cmd
}
