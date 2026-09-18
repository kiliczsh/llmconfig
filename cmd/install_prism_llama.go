package cmd

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/pkg/prismllamacpp"
	"github.com/spf13/cobra"
)

func newInstallPrismLlamaCmd() *cobra.Command {
	var flagBackend string
	var flagFile string
	var flagVersion string

	cmd := &cobra.Command{
		Use:   "prism_llama",
		Short: "Install Prism ML's llama.cpp fork",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer
			if flagFile != "" {
				p.Info("extracting %s...", flagFile)
				if err := prismllamacpp.Extract(flagFile); err != nil {
					return err
				}
				p.Success("Prism llama.cpp installed to %s", prismllamacpp.BinDir())
				return nil
			}

			backend := resolvedBackend(flagBackend, true)
			p.Info("fetching Prism llama.cpp release info...")
			var (
				rel *prismllamacpp.GithubRelease
				err error
			)
			if flagVersion == "" {
				rel, err = prismllamacpp.ReleaseByTag(prismllamacpp.RecommendedReleaseTag)
			} else {
				rel, err = prismllamacpp.ReleaseByTag(flagVersion)
			}
			if err != nil {
				return err
			}
			assets, err := prismllamacpp.PickAssets(rel, backend)
			if err != nil {
				return err
			}
			p.Info("release: %s", rel.TagName)
			p.Info("dest:    %s", prismllamacpp.BinDir())
			for _, asset := range assets {
				p.Info("asset:   %s (%s)", asset.Name, humanize.Bytes(uint64(asset.Size)))
				if err := runBinInstallWithProgress(asset.Name, asset.Size, func(onProgress func(int64, int64)) error {
					return prismllamacpp.Install(asset, onProgress)
				}, func() {}); err != nil {
					return fmt.Errorf("install %s: %w", asset.Name, err)
				}
			}
			p.Success("Prism llama.cpp installed to %s", prismllamacpp.BinDir())
			p.Success("run: llmconfig prism_llama --version")
			return nil
		},
	}
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cuda | metal | cpu (default: auto-detect)")
	cmd.Flags().StringVar(&flagFile, "file", "", "install from a local archive")
	cmd.Flags().StringVar(&flagVersion, "version", "", "install a specific Prism release tag (default: Bonsai-recommended build)")
	return cmd
}
