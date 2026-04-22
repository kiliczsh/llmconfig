package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage model configs",
	}
	cmd.AddCommand(
		newConfigListCmd(),
		newConfigShowCmd(),
		newConfigEditCmd(),
		newConfigPathCmd(),
	)
	return cmd
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all model configs",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			entries, err := os.ReadDir(appCtx.ConfigDir)
			if err != nil {
				if os.IsNotExist(err) {
					p.Info("no configs found (dir: %s)", appCtx.ConfigDir)
					return nil
				}
				return err
			}

			type row struct {
				name string
				mode string
				port string
				path string
			}
			var rows []row
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".yaml")
				fullPath := filepath.Join(appCtx.ConfigDir, e.Name())
				mode, port := "-", "-"
				if cfg, err := config.Load(name, appCtx.ConfigDir); err == nil {
					mode = cfg.Mode
					if cfg.Server.Port > 0 {
						port = fmt.Sprintf("%d", cfg.Server.Port)
					}
				}
				rows = append(rows, row{name, mode, port, fullPath})
			}

			if len(rows) == 0 {
				p.Info("no configs found in %s", appCtx.ConfigDir)
				return nil
			}

			headers := []string{"NAME", "MODE", "PORT", "PATH"}
			tableRows := make([][]string, len(rows))
			for i, r := range rows {
				tableRows[i] = []string{r.name, r.mode, r.port, output.ShortenPath(r.path)}
			}
			p.Table(headers, tableRows)
			return nil
		},
	}
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
				if _, err := exec.LookPath("code"); err == nil {
					editor = "code --wait"
				} else if _, err := exec.LookPath("notepad"); err == nil {
					editor = "notepad"
				} else {
					editor = "vi"
				}
			}

			parts := strings.Fields(editor)
			if len(parts) == 0 {
				return fmt.Errorf("no editor configured (set $EDITOR)")
			}
			editorArgs := append(parts[1:], configPath)
			c := exec.Command(parts[0], editorArgs...)
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
