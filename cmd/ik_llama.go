package cmd

import (
	"fmt"

	"github.com/kiliczsh/llmconfig/pkg/ikllamacpp"
	"github.com/spf13/cobra"
)

func newIkLlamaCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool

	cmd := &cobra.Command{
		Use:   "ik_llama",
		Short: "Show ik_llama.cpp binary status",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagPath {
				path, err := ikllamacpp.FindServer()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}

			if flagVersion {
				path, err := ikllamacpp.FindServer()
				if err != nil {
					return err
				}
				ver, err := ikllamacpp.Version(path)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n%s\n", path, ver)
				return nil
			}

			path, err := ikllamacpp.FindServer()
			if err != nil {
				p.Warn("ik_llama-server not found — run: llmconfig install ik_llama")
				return nil
			}
			ver, err := ikllamacpp.Version(path)
			if err != nil {
				ver = "(version unknown)"
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the ik_llama.cpp version")
	return cmd
}
