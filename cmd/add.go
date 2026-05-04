package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	var flagPath string
	var flagCopy bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Register a local GGUF file as a named model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			if flagPath == "" {
				return fmt.Errorf("model path not provided — check: pass --path <gguf-file>")
			}

			absPath, err := filepath.Abs(flagPath)
			if err != nil {
				return fmt.Errorf("add: resolve path: %w", err)
			}

			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("model file not found at %q — check: verify --path points to an existing GGUF file", absPath)
			}

			modelPath := absPath
			if flagCopy {
				dest := filepath.Join(appCtx.ModelsDir, filepath.Base(absPath))
				if existing, err := os.Stat(dest); err == nil && !os.SameFile(existing, mustStat(absPath)) {
					return fmt.Errorf("models dir already contains %q — check: rename the source file or remove it (llmconfig files clean)", filepath.Base(absPath))
				}
				if err := copyFile(absPath, dest); err != nil {
					return fmt.Errorf("add: copy file: %w", err)
				}
				modelPath = dest
				p.Info("copied to models dir: %s", dest)
			}

			configPath := config.ConfigPath(appCtx.ConfigDir, name)
			if existing, err := config.FindConfigInDir(appCtx.ConfigDir, name); err == nil {
				return fmt.Errorf("config %q already exists at %q — check: choose a different name or run: llmconfig rm %s", name, existing, name)
			}

			cfg := &config.Config{
				Version: 1,
				Name:    name,
				Model: config.ModelSpec{
					Source: "local",
					Path:   modelPath,
				},
			}
			config.ApplyDefaults(cfg)

			content := fmt.Sprintf("version: 1\nname: %s\n\nmodel:\n  source: local\n  path: %s\n\nserver:\n  port: %d\n",
				cfg.Name, modelPath, cfg.Server.Port)

			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("add: write config: %w", err)
			}

			p.Success("added %s → %s", name, configPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagPath, "path", "", "path to the GGUF file (required)")
	cmd.Flags().BoolVar(&flagCopy, "copy", false, "copy file to models dir")
	return cmd
}

// mustStat returns the FileInfo for the given path. Caller has already
// verified existence, so an error here is a TOCTOU between the check and
// the copy; we degrade to nil and let os.SameFile fall through to "not
// same" so the collision guard still fires.
func mustStat(path string) os.FileInfo {
	info, _ := os.Stat(path)
	return info
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
