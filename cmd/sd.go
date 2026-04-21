package cmd

import (
	"fmt"

	"github.com/kiliczsh/llamaconfig/pkg/stablediffusioncpp"
	"github.com/spf13/cobra"
)

func newSdCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool

	cmd := &cobra.Command{
		Use:   "sd",
		Short: "Show stable-diffusion.cpp binary status",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagPath {
				path, err := stablediffusioncpp.FindBinary()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}

			if flagVersion {
				path, err := stablediffusioncpp.FindBinary()
				if err != nil {
					return err
				}
				ver, err := stablediffusioncpp.Version(path)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n%s\n", path, ver)
				return nil
			}

			path, err := stablediffusioncpp.FindBinary()
			if err != nil {
				p.Warn("sd-cli not found — run: llamaconfig install sd")
				return nil
			}
			ver, err := stablediffusioncpp.Version(path)
			if err != nil {
				ver = "(version unknown)"
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the stable-diffusion.cpp version")
	return cmd
}
