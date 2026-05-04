package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llmconfig/pkg/selfupdate"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var flagCheck bool
	var flagVersion string
	var flagForce bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update llmconfig to the latest release",
		Long: `Download a newer llmconfig release from GitHub, verify its checksum, and
replace the running binary in place. The previous binary is kept beside the
new one as <binary>.old in case you need to roll back manually.

Use --check to see whether an update is available without downloading
anything, --version <tag> to install a specific release (including
downgrades), and --force to reinstall even if you're already on the
target version.

The selfupdater only ever fetches assets from
https://github.com/kiliczsh/llmconfig/releases — every other host is refused.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			currentBin, err := os.Executable()
			if err != nil {
				return fmt.Errorf("update: locate current binary: %w", err)
			}
			currentBin, err = filepath.EvalSymlinks(currentBin)
			if err != nil {
				return fmt.Errorf("update: resolve current binary: %w", err)
			}

			var rel *selfupdate.Release
			if flagVersion != "" {
				rel, err = selfupdate.ReleaseByTag(flagVersion)
			} else {
				rel, err = selfupdate.LatestRelease()
			}
			if err != nil {
				return err
			}

			currentClean := strings.TrimPrefix(version, "v")
			latestClean := strings.TrimPrefix(rel.TagName, "v")

			if flagCheck {
				switch {
				case currentClean == "dev":
					fmt.Printf("running a dev build; latest release is %s\n", rel.TagName)
				case currentClean == latestClean:
					fmt.Printf("you are up to date (%s)\n", rel.TagName)
				default:
					fmt.Printf("update available: %s → %s\n", versionTag(currentClean), rel.TagName)
					fmt.Println("run: llmconfig update")
				}
				return nil
			}

			if currentClean == latestClean && !flagForce {
				p.Success("already on %s (use --force to reinstall)", rel.TagName)
				return nil
			}

			asset, err := selfupdate.PickAsset(rel)
			if err != nil {
				return err
			}

			expectedSHA, err := selfupdate.FetchChecksum(rel, asset.Name)
			if err != nil && !errors.Is(err, selfupdate.ErrNoChecksum) {
				return err
			}
			if expectedSHA == "" {
				return fmt.Errorf("update: no checksum entry for %s in checksums.txt — refusing to install unverified binary", asset.Name)
			}

			p.Info("release: %s", rel.TagName)
			p.Info("asset:   %s (%s)", asset.Name, humanize.Bytes(uint64(asset.Size)))
			p.Info("dest:    %s", currentBin)

			// Stage the download next to the target binary so the final
			// rename is same-filesystem (cross-fs renames fail on Windows
			// and on Linux when /tmp is tmpfs).
			binDir := filepath.Dir(currentBin)
			archivePath := filepath.Join(binDir, ".llmconfig-update-"+asset.Name)
			defer os.Remove(archivePath)

			err = runBinInstallWithProgress(asset.Name, asset.Size, func(onProgress func(int64, int64)) error {
				return selfupdate.DownloadAsset(asset, archivePath, expectedSHA, onProgress)
			}, func() {
				p.Success("downloaded and verified %s", asset.Name)
			})
			if err != nil {
				return fmt.Errorf("update: %w", err)
			}

			extracted, err := selfupdate.ExtractRelease(archivePath)
			if err != nil {
				return fmt.Errorf("update: %w", err)
			}
			defer os.RemoveAll(extracted.StageDir)

			// Move the extracted binary next to the current one so Swap()
			// can rename within a single filesystem.
			stagedNew := currentBin + ".new"
			if err := moveFile(extracted.Llmconfig, stagedNew); err != nil {
				return fmt.Errorf("update: stage new binary: %w", err)
			}
			if err := selfupdate.Swap(currentBin, stagedNew); err != nil {
				_ = os.Remove(stagedNew)
				return fmt.Errorf("update: %w", err)
			}

			// Best-effort llmc swap. Failure here is loud-warned but doesn't
			// fail the command — the primary binary is already updated.
			if extracted.Llmc != "" {
				llmcName := "llmc"
				if runtime.GOOS == "windows" {
					llmcName = "llmc.exe"
				}
				llmcTarget := filepath.Join(binDir, llmcName)
				if _, err := os.Stat(llmcTarget); err == nil {
					stagedLlmc := llmcTarget + ".new"
					if err := moveFile(extracted.Llmc, stagedLlmc); err != nil {
						p.Warn("llmc not updated (%v) — primary binary is on %s", err, rel.TagName)
					} else if err := selfupdate.Swap(llmcTarget, stagedLlmc); err != nil {
						_ = os.Remove(stagedLlmc)
						p.Warn("llmc not updated (%v) — primary binary is on %s", err, rel.TagName)
					} else {
						p.Success("llmc updated alongside")
					}
				}
			}

			p.Success("updated %s → %s", versionTag(currentClean), rel.TagName)
			p.Info("previous binary kept as %s.old", currentBin)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagCheck, "check", false, "check for a newer release without installing")
	cmd.Flags().StringVar(&flagVersion, "version", "", "install a specific release tag (default: latest)")
	cmd.Flags().BoolVar(&flagForce, "force", false, "reinstall even if already on the target version")
	return cmd
}

// moveFile renames src → dst, falling back to copy+remove for cross-device
// situations the renamer can't handle (e.g. /tmp on a separate filesystem).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dst, 0755)
	}
	_ = os.Remove(src)
	return nil
}

func versionTag(s string) string {
	if s == "" || s == "dev" {
		return s
	}
	if strings.HasPrefix(s, "v") {
		return s
	}
	return "v" + s
}
