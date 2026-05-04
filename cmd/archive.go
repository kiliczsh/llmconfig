package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/internal/archive"
	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/spf13/cobra"
)

func newArchiveCmd() *cobra.Command {
	var flagOutput string
	var flagAll bool
	var flagForce bool

	cmd := &cobra.Command{
		Use:   "archive [name...]",
		Short: "Bundle one or more models (config + cached GGUF) into a .llmcpkg file",
		Long: `Bundle one or more models into a .llmcpkg file.

With no names, an interactive multi-select is shown. Pass --all to
skip the selector and include every model with a config.

.llmcpkg is an uncompressed POSIX tar; any tar-compatible tool can
unpack it manually ("tar -xf foo.llmcpkg").`,
		Example: `  llmconfig archive                       # interactive selector
  llmconfig archive gemma-4-e2b
  llmconfig archive gemma-4-e2b qwen-4b -o models.llmcpkg
  llmconfig archive --all -o backup.llmcpkg`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			available, err := listConfigNames(appCtx.ConfigDir)
			if err != nil {
				return err
			}
			if len(available) == 0 {
				return fmt.Errorf("no configs found in %s", appCtx.ConfigDir)
			}

			var selected []string
			switch {
			case flagAll:
				selected = available
			case len(args) > 0:
				for _, name := range args {
					if !contains(available, name) {
						return fmt.Errorf("config %q not found in %s", name, appCtx.ConfigDir)
					}
				}
				selected = args
			default:
				selected, err = pickModelsToArchive(available, appCtx.ConfigDir)
				if err != nil {
					return err
				}
				if len(selected) == 0 {
					p.Info("nothing selected")
					return nil
				}
			}

			// Resolve output path. Auto-compute a reasonable default when the
			// user didn't supply -o.
			outPath := flagOutput
			if outPath == "" {
				outPath = defaultArchivePath(selected)
				if len(args) == 0 && !flagAll {
					if err := runForm(huh.NewForm(huh.NewGroup(
						huh.NewInput().
							Title("Output path").
							Value(&outPath),
					))); err != nil {
						return err
					}
				}
			}
			outPath, err = filepath.Abs(outPath)
			if err != nil {
				return err
			}

			if _, err := os.Stat(outPath); err == nil && !flagForce {
				return fmt.Errorf("%s already exists — pass --force to overwrite", outPath)
			}

			// Build CreateEntries.
			entries, totalBytes, err := buildCreateEntries(selected, appCtx.ConfigDir, appCtx.ModelsDir, p)
			if err != nil {
				return err
			}

			p.Info("archiving %d model(s), approx %s → %s", len(entries), humanize.Bytes(uint64(totalBytes)), outPath)

			exportedBy := fmt.Sprintf("llmconfig %s", version)
			progress := newArchiveProgress()
			if err := archive.Create(outPath, entries, exportedBy, progress.update); err != nil {
				_ = os.Remove(outPath)
				return err
			}
			progress.finish()

			info, _ := os.Stat(outPath)
			var size string
			if info != nil {
				size = humanize.Bytes(uint64(info.Size()))
			}
			p.Success("wrote %s (%s)", outPath, size)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "output path (default: <name>.llmcpkg in cwd)")
	cmd.Flags().BoolVar(&flagAll, "all", false, "archive every configured model")
	cmd.Flags().BoolVarP(&flagForce, "force", "f", false, "overwrite the output file if it exists")
	return cmd
}

func listConfigNames(configDir string) ([]string, error) {
	if _, err := os.Stat(configDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return config.ListConfigNames(configDir)
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func pickModelsToArchive(available []string, configDir string) ([]string, error) {
	opts := make([]huh.Option[string], 0, len(available))
	for _, name := range available {
		label := name
		if cfg, err := config.Load(name, configDir); err == nil {
			if cfg.Model.File != "" {
				label = fmt.Sprintf("%s  (%s)", name, cfg.Model.File)
			}
		}
		opts = append(opts, huh.NewOption(label, name))
	}

	var selected []string
	err := runForm(huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Select models to archive").
			Description("Space to toggle, enter to confirm").
			Options(opts...).
			Value(&selected).
			Filterable(true),
	)))
	if err != nil {
		return nil, err
	}
	return selected, nil
}

func buildCreateEntries(names []string, configDir, modelsDir string, p interface{ Warn(string, ...any) }) ([]archive.CreateEntry, int64, error) {
	var entries []archive.CreateEntry
	var total int64
	for _, name := range names {
		cfgPath, err := config.FindConfigInDir(configDir, name)
		if err != nil {
			return nil, 0, fmt.Errorf("locate %s: %w", name, err)
		}
		cfg, err := config.LoadFile(cfgPath)
		if err != nil {
			return nil, 0, fmt.Errorf("load %s: %w", name, err)
		}
		config.ApplyDefaults(cfg)

		entry := archive.CreateEntry{
			Name:       name,
			ConfigPath: cfgPath,
			Source:     cfg.Model.Source,
		}

		// Locate the model file. Prefer downloaded copy; fall back to explicit
		// local path. Missing files are warned about but don't block the
		// archive (someone may just want the config).
		var modelPath string
		switch cfg.Model.Source {
		case "local":
			modelPath = cfg.Model.Path
		default:
			if cfg.Model.File != "" {
				modelPath = filepath.Join(modelsDir, cfg.Model.File)
			}
		}
		switch {
		case modelPath == "":
			p.Warn("%s: config has no model file reference — archiving config only", name)
		default:
			if info, err := os.Stat(modelPath); err == nil {
				entry.ModelPath = modelPath
				total += info.Size()
			} else {
				p.Warn("%s: model file not found (%s) — archiving config only", name, modelPath)
			}
		}

		entries = append(entries, entry)
	}
	return entries, total, nil
}

func defaultArchivePath(names []string) string {
	cwd, _ := os.Getwd()
	if len(names) == 1 {
		return filepath.Join(cwd, names[0]+".llmcpkg")
	}
	return filepath.Join(cwd, fmt.Sprintf("llmconfig-%s.llmcpkg", time.Now().UTC().Format("20060102")))
}

// archiveProgress rate-limits carriage-returned progress lines so a 40GB
// copy doesn't flood the terminal.
type archiveProgress struct {
	lastEntry string
	shown     bool
}

func newArchiveProgress() *archiveProgress { return &archiveProgress{} }

func (a *archiveProgress) update(entry string, copied, total int64) {
	if entry != a.lastEntry {
		if a.shown {
			fmt.Println()
		}
		a.lastEntry = entry
		a.shown = true
	}
	if total > 0 {
		pct := float64(copied) / float64(total) * 100
		fmt.Printf("\r  %s  %s / %s  (%.0f%%)  ",
			entry, humanize.Bytes(uint64(copied)), humanize.Bytes(uint64(total)), pct)
	} else {
		fmt.Printf("\r  %s  %s  ", entry, humanize.Bytes(uint64(copied)))
	}
}

func (a *archiveProgress) finish() {
	if a.shown {
		fmt.Println()
	}
}
