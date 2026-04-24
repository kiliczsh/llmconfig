package cmd

import (
	"fmt"

	"github.com/kiliczsh/llmconfig/pkg/llamacpp"
	"github.com/spf13/cobra"
)

func newLlamaCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool

	cmd := &cobra.Command{
		Use:   "llama",
		Short: "Show llama.cpp binary status",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagPath {
				path, err := llamacpp.FindServer()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}

			if flagVersion {
				path, err := llamacpp.FindServer()
				if err != nil {
					return err
				}
				ver, err := llamacpp.Version(path)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n%s\n", path, ver)
				return nil
			}

			path, err := llamacpp.FindServer()
			if err != nil {
				p.Warn("llama-server not found — run: llmconfig install llama")
				return nil
			}
			ver, err := llamacpp.Version(path)
			if err != nil {
				ver = "(version unknown)"
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the llama.cpp version")
	return cmd
}
