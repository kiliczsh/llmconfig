package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llamaconfig/internal/dirs"
	"github.com/spf13/cobra"
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
		Short:   "List all cached GGUF files with sizes",
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
		if !strings.HasSuffix(e.Name(), ".gguf") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, cacheEntry{
			name: e.Name(),
			path: filepath.Join(cacheDir, e.Name()),
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
			huh.NewForm(huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Remove %d file(s)?", len(toRemove))).
					Value(&confirm),
			)).Run()

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

// referencedFiles returns a set of GGUF filenames mentioned in any config.
func referencedFiles(configDir string) map[string]bool {
	refs := map[string]bool{}
	pattern := filepath.Join(configDir, "*.yaml")
	matches, _ := filepath.Glob(pattern)
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Simple scan — look for "file: <name>.gguf"
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "file:") {
				name := strings.TrimSpace(strings.TrimPrefix(line, "file:"))
				if strings.HasSuffix(name, ".gguf") {
					refs[name] = true
				}
			}
		}
	}
	return refs
}
