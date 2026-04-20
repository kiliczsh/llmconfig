package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) {
	version = v
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print llamaconfig version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("llamaconfig %s\n", version)
			return nil
		},
	}
}
