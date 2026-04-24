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

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the model file cache",
	}
	cmd.AddCommand(
		newCacheLsCmd(),
		newCacheCleanCmd(),
		newCachePathCmd(),
	)
	return cmd
}

func newCachePathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the cache directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(dirs.CacheDir())
			return nil
		},
	}
}

func newCacheLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all cached model files with sizes",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			entries, total, err := listCachedFiles(dirs.CacheDir())
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				p.Info("cache is empty (%s)", dirs.CacheDir())
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

type cacheEntry struct {
	name string
	path string
	size int64
}

func listCachedFiles(cacheDir string) ([]cacheEntry, int64, error) {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	var files []cacheEntry
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// Skip in-progress download temp files so they don't confuse
		// listing/clean output.
		name := e.Name()
		if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".part") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, cacheEntry{
			name: name,
			path: filepath.Join(cacheDir, name),
			size: info.Size(),
		})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, total, nil
}

func newCacheCleanCmd() *cobra.Command {
	var flagAll bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove cached files not linked to any config",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			cacheDir := dirs.CacheDir()

			entries, _, err := listCachedFiles(cacheDir)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				p.Info("cache is empty")
				return nil
			}

			var toRemove []cacheEntry
			if flagAll {
				toRemove = entries
			} else {
				// Find files referenced by at least one config
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
			form := huh.NewForm(huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Remove %d file(s)?", len(toRemove))).
					Value(&confirm),
			))
			if err := form.Run(); err != nil {
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

	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "remove all cached files")
	return cmd
}

// referencedFiles returns the set of cached filenames that at least one
// config points at — main model, draft model, and multimodal projector
// file. The YAML is parsed (not scanned line-by-line) so nested structs
// and non-gguf backends (whisper .bin, SD weights) are picked up.
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
