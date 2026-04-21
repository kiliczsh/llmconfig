package cmd

import (
	"fmt"

	"github.com/kiliczsh/llamaconfig/pkg/whispercpp"
	"github.com/spf13/cobra"
)

func newWhisperCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool

	cmd := &cobra.Command{
		Use:   "whisper",
		Short: "Show whisper.cpp binary status",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagPath {
				path, err := whispercpp.FindBinary()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}

			if flagVersion {
				path, err := whispercpp.FindBinary()
				if err != nil {
					return err
				}
				ver, err := whispercpp.Version(path)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n%s\n", path, ver)
				return nil
			}

			path, err := whispercpp.FindBinary()
			if err != nil {
				p.Warn("whisper-cli not found — run: llamaconfig install whisper")
				return nil
			}
			ver, err := whispercpp.Version(path)
			if err != nil {
				ver = "(version unknown)"
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the whisper.cpp version")
	return cmd
}
