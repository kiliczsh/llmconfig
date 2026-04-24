package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/output"
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
		Use:               "show [name]",
		Short:             "Print the resolved config (with defaults applied)",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeConfigNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			var nameArg string
			if len(args) > 0 {
				nameArg = args[0]
			}
			name, err := pickConfig(nameArg, appCtx.ConfigDir)
			if err != nil {
				return err
			}
			p := appCtx.Printer

			cfg, err := config.Load(name, appCtx.ConfigDir)
			if err != nil {
				return err
			}

			if !flagRaw {
				config.ApplyDefaults(cfg)
				warnDeprecated(cfg, p)
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
		Use:               "edit [name]",
		Short:             "Open config in $EDITOR",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeConfigNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			var nameArg string
			if len(args) > 0 {
				nameArg = args[0]
			}
			name, err := pickConfig(nameArg, appCtx.ConfigDir)
			if err != nil {
				return err
			}

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

			parts := splitEditorCmd(editor)
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
		Use:               "path [name]",
		Short:             "Print the config file path",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeConfigNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			var nameArg string
			if len(args) > 0 {
				nameArg = args[0]
			}
			name, err := pickConfig(nameArg, appCtx.ConfigDir)
			if err != nil {
				return err
			}
			configPath := filepath.Join(appCtx.ConfigDir, name+".yaml")
			fmt.Println(configPath)
			return nil
		},
	}
}

// splitEditorCmd is a whitespace tokenizer that honors double/single
// quotes, so `EDITOR='"C:\Program Files\VS Code\Code.exe" --wait'` splits
// into two tokens instead of shattering the path at every space.
func splitEditorCmd(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	var quoteChar rune
	for _, r := range s {
		switch {
		case inQuote && r == quoteChar:
			inQuote = false
		case !inQuote && (r == '"' || r == '\''):
			inQuote = true
			quoteChar = r
		case !inQuote && (r == ' ' || r == '\t'):
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func warnDeprecated(cfg *config.Config, p *output.Printer) {
	// No fields are currently deprecated. When a field is renamed or
	// removed, emit p.Warn("<yaml.path> is deprecated: use <replacement>")
	// here so that `config show` surfaces migration hints.
}
