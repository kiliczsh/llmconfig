package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

// SetBuildInfo is called from main with values injected at link time
// (-ldflags "-X main.version=... -X main.commit=... -X main.date=...").
// When fields are empty, values are read from debug.BuildInfo so `go install`
// builds still show useful detail.
func SetBuildInfo(v, c, d string) {
	if v != "" {
		version = v
	}
	if c != "" {
		commit = c
	}
	if d != "" {
		date = d
	}
}

// SetVersion is retained for backwards compatibility with older main.go.
func SetVersion(v string) { SetBuildInfo(v, "", "") }

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print llamaconfig version",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, d := commit, date
			if c == "" || d == "" {
				if info, ok := debug.ReadBuildInfo(); ok {
					for _, s := range info.Settings {
						if c == "" && s.Key == "vcs.revision" {
							c = s.Value
						}
						if d == "" && s.Key == "vcs.time" {
							d = s.Value
						}
					}
				}
			}
			if len(c) > 12 {
				c = c[:12]
			}
			fmt.Printf("llamaconfig %s", version)
			if c != "" {
				fmt.Printf(" (%s", c)
				if d != "" {
					fmt.Printf(", %s", d)
				}
				fmt.Printf(")")
			}
			fmt.Println()
			return nil
		},
	}
}
