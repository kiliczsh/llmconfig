package cmd

import (
	"fmt"

	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/hardware"
	"github.com/kiliczsh/llmconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var flagProfile string

	cmd := &cobra.Command{
		Use:   "inspect [name]",
		Short: "Show the llama.cpp command that would be run",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			var arg string
			if len(args) > 0 {
				arg = args[0]
			}
			name, err := pickConfig(arg, appCtx.ConfigDir)
			if err != nil {
				return err
			}

			cfg, err := config.Load(name, appCtx.ConfigDir)
			if err != nil {
				return err
			}
			config.ApplyDefaults(cfg)
			if err := config.Validate(cfg); err != nil {
				return err
			}

			var hw *hardware.DetectionResult
			if flagProfile != "" {
				hw = profileOverride(flagProfile)
			} else {
				hw = hardware.Detect()
			}

			binaryPath, err := resolveBackendBinary(cfg.Backend)
			if err != nil {
				return err
			}

			rc, err := config.Resolve(cfg, hw, binaryPath)
			if err != nil {
				return err
			}

			if cfg.Mode == "interactive" {
				cliBin := runner.DeriveCLIBinary(binaryPath, cfg.Backend)
				fmt.Println(runner.FormatInteractiveArgs(cliBin, rc))
			} else {
				fmt.Println(runner.FormatArgs(binaryPath, rc))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagProfile, "profile", "", "inspect for a specific hardware profile")
	return cmd
}
