package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

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
	var flagCheck bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print llmconfig version",
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
			fmt.Printf("llmconfig %s", version)
			if c != "" {
				fmt.Printf(" (%s", c)
				if d != "" {
					fmt.Printf(", %s", d)
				}
				fmt.Printf(")")
			}
			fmt.Println()

			if flagCheck {
				latest, err := fetchLatestVersion()
				if err != nil {
					fmt.Printf("  could not check for updates: %v\n", err)
					return nil
				}
				current := strings.TrimPrefix(version, "v")
				latestClean := strings.TrimPrefix(latest, "v")
				if current == "dev" || current == latestClean {
					fmt.Printf("  you are up to date (%s)\n", latest)
				} else {
					fmt.Printf("  new version available: %s\n", latest)
					fmt.Printf("  update: irm https://raw.githubusercontent.com/kiliczsh/llmconfig/main/install.ps1 | iex\n")
					fmt.Printf("     or:  curl -fsSL https://raw.githubusercontent.com/kiliczsh/llmconfig/main/install.sh | bash\n")
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagCheck, "check", false, "check for a newer release")
	return cmd
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/kiliczsh/llmconfig/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		TagName string `json:"tag_name"`
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.TagName == "" {
		return "", fmt.Errorf("no releases found")
	}
	return result.TagName, nil
}
