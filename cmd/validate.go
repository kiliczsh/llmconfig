package cmd

import (
	"fmt"

	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var flagFile string

	cmd := &cobra.Command{
		Use:   "validate [name]",
		Short: "Validate a config file without running it",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			var cfg *config.Config
			var err error

			if flagFile != "" {
				cfg, err = config.LoadFile(flagFile)
			} else if len(args) == 1 {
				cfg, err = config.Load(args[0], appCtx.ConfigDir)
			} else {
				return fmt.Errorf("provide a model name or --file <path>")
			}
			if err != nil {
				return err
			}

			config.ApplyDefaults(cfg)
			if err := config.Validate(cfg); err != nil {
				p.Error("invalid: %v", err)
				return err
			}

			p.Success("config is valid: %s (version %d, source: %s)", cfg.Name, cfg.Version, cfg.Model.Source)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagFile, "file", "", "validate a specific file instead of a named model")
	return cmd
}
