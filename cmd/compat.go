package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/internal/bench"
	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/dirs"
	"github.com/kiliczsh/llmconfig/internal/hardware"
	"github.com/spf13/cobra"
)

func bandwidthGBs(hw *hardware.DetectionResult) float64 {
	switch hw.Class {
	case hardware.ClassAppleSilicon:
		return 200
	case hardware.ClassNVIDIA:
		return 500
	case hardware.ClassAMD:
		return 350
	default:
		return 50
	}
}

func newCompatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compat",
		Short: "Show which models fit in RAM and inference speed",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			hw := hardware.Detect()
			bwGBs := bandwidthGBs(hw)
			ramGB := float64(hw.RAMBytes) / (1 << 30)
			availableGB := ramGB - 2.5

			ramLabel := "unknown"
			if hw.RAMBytes > 0 {
				ramLabel = humanize.Bytes(hw.RAMBytes)
			}
			fmt.Printf("Hardware: %s — %s — ~%.0f GB/s\n\n", hw.GPUName, ramLabel, bwGBs)

			pattern := filepath.Join(appCtx.ConfigDir, "*.yaml")
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return err
			}
			sort.Strings(matches)

			type row struct {
				name   string
				size   string
				sizeGB float64
				genTPS string
				fits   string
				compat string
			}

			var rows []row
			for _, path := range matches {
				cfg, err := config.LoadFile(path)
				if err != nil {
					continue
				}
				config.ApplyDefaults(cfg)

				cacheFile := filepath.Join(dirs.CacheDir(), cfg.Model.File)
				if cfg.Model.Source == "local" {
					cacheFile = cfg.Model.Path
				}

				var sizeGB float64
				var sizeLabel string
				if info, err := os.Stat(cacheFile); err == nil {
					sizeGB = float64(info.Size()) / (1 << 30)
					sizeLabel = humanize.Bytes(uint64(info.Size()))
				} else {
					sizeLabel = "not cached"
				}

				var genTPS, fits, compat string

				// Real bench result takes priority; fall back to estimate
				if r, err := bench.Load(dirs.BenchDir(), cfg.Name); err == nil {
					genTPS = fmt.Sprintf("%.1f t/s", r.AvgGenerateTPS)
				} else if sizeGB > 0 {
					genTPS = fmt.Sprintf("~%.0f t/s", bwGBs/sizeGB)
				} else {
					genTPS = "n/a"
				}

				if sizeGB > 0 && availableGB > 0 {
					if sizeGB <= availableGB {
						fits = "yes"
						estTPS := bwGBs / sizeGB
						if estTPS >= 20 {
							compat = "YES"
						} else {
							compat = "slow"
						}
					} else {
						fits = "no (OOM)"
						compat = "NO"
					}
				} else {
					fits = "?"
					compat = "?"
				}

				rows = append(rows, row{
					name:   cfg.Name,
					size:   sizeLabel,
					sizeGB: sizeGB,
					genTPS: genTPS,
					fits:   fits,
					compat: compat,
				})
			}

			if len(rows) == 0 {
				p.Info("no model configs found")
				return nil
			}

			headers := []string{"NAME", "SIZE", "GEN TOK/S", "RAM FIT", "COMPATIBLE"}
			tableRows := make([][]string, len(rows))
			for i, r := range rows {
				tableRows[i] = []string{r.name, r.size, r.genTPS, r.fits, r.compat}
			}
			p.Table(headers, tableRows)
			return nil
		},
	}
	return cmd
}
