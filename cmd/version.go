package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.1.0"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print llamaconfig version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("llamaconfig %s\n", Version)
			return nil
		},
	}
}
