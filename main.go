package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/kiliczsh/llmconfig/cmd"
)

// These are set at link time via -ldflags:
//
//	-X main.version=v1.2.3 -X main.commit=abc123 -X main.date=2026-04-22T12:00:00Z
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	cmd.SetBuildInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, cmd.ErrAborted) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
