package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/internal/dirs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Manage downloaded model files",
	}
	cmd.AddCommand(
		newFilesLsCmd(),
		newFilesCleanCmd(),
		newFilesPathCmd(),
	)
	return cmd
}

func newFilesPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the models directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(dirs.ModelsDir())
			return nil
		},
	}
}

func newFilesLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all downloaded model files with sizes",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			entries, total, err := listModelFiles(dirs.ModelsDir())
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				p.Info("models dir is empty (%s)", dirs.ModelsDir())
				return nil
			}

			headers := []string{"FILE", "SIZE"}
			rows := make([][]string, len(entries))
			for i, e := range entries {
				rows[i] = []string{e.name, humanize.Bytes(uint64(e.size))}
			}
			p.Table(headers, rows)
			fmt.Printf("\n  Total: %s  (%d files)\n", humanize.Bytes(uint64(total)), len(entries))
			return nil
		},
	}
}

type modelFileEntry struct {
	name string
	path string
	size int64
}

func listModelFiles(modelsDir string) ([]modelFileEntry, int64, error) {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	var files []modelFileEntry
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".part") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, modelFileEntry{
			name: name,
			path: filepath.Join(modelsDir, name),
			size: info.Size(),
		})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, total, nil
}

func newFilesCleanCmd() *cobra.Command {
	var flagAll bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove model files not linked to any config",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			modelsDir := dirs.ModelsDir()

			entries, _, err := listModelFiles(modelsDir)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				p.Info("models dir is empty")
				return nil
			}

			var toRemove []modelFileEntry
			if flagAll {
				toRemove = entries
			} else {
				referenced := referencedFiles(appCtx.ConfigDir)
				for _, e := range entries {
					if !referenced[e.name] {
						toRemove = append(toRemove, e)
					}
				}
			}

			if len(toRemove) == 0 {
				p.Info("no unreferenced files to clean")
				return nil
			}

			var totalSize int64
			for _, e := range toRemove {
				totalSize += e.size
				fmt.Printf("  %s  (%s)\n", e.name, humanize.Bytes(uint64(e.size)))
			}
			fmt.Printf("\n  Will free %s\n\n", humanize.Bytes(uint64(totalSize)))

			var confirm bool
			if err := runForm(huh.NewForm(huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Remove %d file(s)?", len(toRemove))).
					Value(&confirm),
			))); err != nil {
				return err
			}
			if !confirm {
				p.Info("cancelled")
				return nil
			}

			for _, e := range toRemove {
				if err := os.Remove(e.path); err != nil {
					p.Error("remove %s: %v", e.name, err)
				} else {
					p.Success("removed %s", e.name)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "remove all downloaded model files")
	return cmd
}

func referencedFiles(configDir string) map[string]bool {
	type fileRef struct {
		File string `yaml:"file"`
	}
	type modelRef struct {
		File   string   `yaml:"file"`
		Draft  *fileRef `yaml:"draft,omitempty"`
		MMProj *fileRef `yaml:"mmproj,omitempty"`
	}
	type minimal struct {
		Model modelRef `yaml:"model"`
	}

	refs := map[string]bool{}
	matches, _ := filepath.Glob(filepath.Join(configDir, "*.yaml"))
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m minimal
		if err := yaml.Unmarshal(data, &m); err != nil {
			continue
		}
		if m.Model.File != "" {
			refs[filepath.Base(m.Model.File)] = true
		}
		if m.Model.Draft != nil && m.Model.Draft.File != "" {
			refs[filepath.Base(m.Model.Draft.File)] = true
		}
		if m.Model.MMProj != nil && m.Model.MMProj.File != "" {
			refs[filepath.Base(m.Model.MMProj.File)] = true
		}
	}
	return refs
}
