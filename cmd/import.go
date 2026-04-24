package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/internal/archive"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var flagOverwrite bool
	var flagYes bool

	cmd := &cobra.Command{
		Use:     "import <file.llmcpkg>",
		Aliases: []string{"restore"},
		Short:   "Extract a .llmcpkg bundle into the config and cache directories",
		Args:    cobra.ExactArgs(1),
		Example: `  llmconfig import gemma-4-e2b.llmcpkg
  llmconfig import backup.llmcpkg --overwrite`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			archivePath, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			if _, err := os.Stat(archivePath); err != nil {
				return fmt.Errorf("archive not found: %s", archivePath)
			}

			// Read the manifest first so the user sees what's inside before
			// anything is written.
			manifest, err := archive.Open(archivePath)
			if err != nil {
				return err
			}

			printManifestSummary(p, manifest, appCtx.ConfigDir, appCtx.CacheDir)

			// Confirm unless --yes.
			if !flagYes {
				var confirm bool
				if err := huh.NewForm(huh.NewGroup(
					huh.NewConfirm().
						Title(fmt.Sprintf("Import %d entry(ies)?", len(manifest.Entries))).
						Value(&confirm),
				)).Run(); err != nil {
					return err
				}
				if !confirm {
					p.Info("cancelled")
					return nil
				}
			}

			progress := newArchiveProgress()
			result, err := archive.Extract(archivePath, appCtx.ConfigDir, appCtx.CacheDir, flagOverwrite, progress.update)
			if err != nil {
				progress.finish()
				return err
			}
			progress.finish()

			for _, name := range result.Installed {
				p.Success("imported %s", name)
			}
			for _, name := range result.Skipped {
				p.Warn("skipped %s (already exists — pass --overwrite to replace)", name)
			}
			if len(result.Installed) == 0 && len(result.Skipped) == 0 {
				p.Info("archive was empty")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagOverwrite, "overwrite", false, "replace existing configs and cached files on conflict")
	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func printManifestSummary(p interface {
	Info(string, ...any)
}, m *archive.Manifest, configDir, cacheDir string) {
	p.Info("archive exported by %s at %s", m.ExportedBy, m.ExportedAt.Local().Format("2006-01-02 15:04"))
	fmt.Println()
	fmt.Printf("  %-24s %-10s %s\n", "NAME", "SIZE", "STATUS")
	for _, e := range m.Entries {
		size := "-"
		if e.Size > 0 {
			size = humanize.Bytes(uint64(e.Size))
		}
		status := "new"
		cfgDest := filepath.Join(configDir, e.Name+".yaml")
		if _, err := os.Stat(cfgDest); err == nil {
			status = "exists (config)"
		}
		if e.ModelFile != "" {
			modelDest := filepath.Join(cacheDir, filepath.Base(e.ModelFile))
			if _, err := os.Stat(modelDest); err == nil {
				if status == "new" {
					status = "exists (model)"
				} else {
					status = "exists (both)"
				}
			}
		}
		fmt.Printf("  %-24s %-10s %s\n", e.Name, size, status)
	}
	fmt.Println()
}
