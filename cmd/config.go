package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage model configs",
	}
	cmd.AddCommand(
		newConfigShowCmd(),
		newConfigEditCmd(),
		newConfigPathCmd(),
	)
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var flagRaw bool

	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Print the resolved config (with defaults applied)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			appCtx := appCtxFrom(cmd.Context())

			cfg, err := config.Load(name, appCtx.ConfigDir)
			if err != nil {
				return err
			}

			if !flagRaw {
				config.ApplyDefaults(cfg)
			}

			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("config show: marshal: %w", err)
			}
			fmt.Print(string(data))
			return nil
		},
	}
	cmd.Flags().BoolVar(&flagRaw, "raw", false, "print raw YAML without defaults")
	return cmd
}

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Open config in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			appCtx := appCtxFrom(cmd.Context())

			configPath := filepath.Join(appCtx.ConfigDir, name+".yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return fmt.Errorf("config %q not found at %s", name, configPath)
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				// Fallback by platform
				if _, err := exec.LookPath("code"); err == nil {
					editor = "code --wait"
				} else if _, err := exec.LookPath("notepad"); err == nil {
					editor = "notepad"
				} else {
					editor = "vi"
				}
			}

			c := exec.Command("sh", "-c", editor+" "+configPath)
			if os.Getenv("EDITOR") != "" {
				c = exec.Command(editor, configPath)
			}
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path <name>",
		Short: "Print the config file path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			appCtx := appCtxFrom(cmd.Context())
			configPath := filepath.Join(appCtx.ConfigDir, name+".yaml")
			fmt.Println(configPath)
			return nil
		},
	}
}
