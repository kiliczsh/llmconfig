package cmd

import (
	"fmt"

	"github.com/kiliczsh/llmconfig/pkg/prismllamacpp"
	"github.com/spf13/cobra"
)

func newPrismLlamaCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool

	cmd := &cobra.Command{
		Use:   "prism_llama",
		Short: "Show Prism ML llama.cpp binary status",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer
			path, err := prismllamacpp.FindServer()
			if err != nil {
				if flagPath || flagVersion {
					return err
				}
				p.Warn("Prism llama-server not found — install a Prism ML release in %s", prismllamacpp.BinDir())
				return nil
			}

			if flagPath {
				fmt.Println(path)
				return nil
			}
			ver, versionErr := prismllamacpp.Version(path)
			if versionErr != nil {
				if flagVersion {
					return versionErr
				}
				ver = "(version unknown)"
			}
			if flagVersion {
				fmt.Printf("%s\n%s\n", path, ver)
				return nil
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the Prism llama.cpp version")
	return cmd
}
