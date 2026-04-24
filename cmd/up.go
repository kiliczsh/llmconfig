package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/dirs"
	"github.com/kiliczsh/llamaconfig/internal/downloader"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
	"github.com/kiliczsh/llamaconfig/internal/runner"
	"github.com/kiliczsh/llamaconfig/internal/state"
	"github.com/kiliczsh/llamaconfig/pkg/llamacpp"
	"github.com/kiliczsh/llamaconfig/pkg/stablediffusioncpp"
	"github.com/kiliczsh/llamaconfig/pkg/whispercpp"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	var flagPort int
	var flagProfile string
	var flagDryRun bool
	var flagNoDownload bool

	cmd := &cobra.Command{
		Use:   "up [name]",
		Short: "Start a model from its config file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			var nameArg string
			if len(args) > 0 {
				nameArg = args[0]
			}
			name, err := pickConfig(nameArg, appCtx.ConfigDir)
			if err != nil {
				return err
			}
			p := appCtx.Printer

			// Load and validate config
			cfg, err := config.Load(name, appCtx.ConfigDir)
			if err != nil {
				return err
			}
			config.ApplyDefaults(cfg)
			if err := config.Validate(cfg); err != nil {
				return err
			}

			// Override port if specified
			if flagPort > 0 {
				cfg.Server.Port = flagPort
			}

			// Check already running
			existing, err := appCtx.StateStore.Get(name)
			if err != nil {
				return err
			}
			if existing != nil && existing.Status == "running" {
				r := runner.New()
				if r.IsAlive(existing) {
					p.Success("%s is already running at http://%s:%d (PID %d)", name, existing.Host, existing.Port, existing.PID)
					return nil
				}
			}

			// Detect hardware
			var hw *hardware.DetectionResult
			if flagProfile != "" {
				hw = profileOverride(flagProfile)
			} else {
				hw = hardware.Detect()
			}

			// Select binary based on backend. In dry-run we fall back to a
			// placeholder name so `up --dry-run` still prints a usable
			// command even on machines where the backend isn't installed.
			binaryPath, err := resolveBackendBinary(cfg.Backend)
			if err != nil {
				if !flagDryRun {
					return err
				}
				binaryPath = placeholderBinaryName(cfg.Backend)
			}

			// Resolve config → RunConfig
			rc, err := config.Resolve(cfg, hw, binaryPath)
			if err != nil {
				return err
			}

			// Dry run: print command and exit (no binary or model needed)
			if flagDryRun {
				if cfg.Mode == "interactive" {
					cliBin := runner.DeriveCLIBinary(binaryPath, cfg.Backend)
					fmt.Println(runner.FormatInteractiveArgs(cliBin, rc))
				} else {
					fmt.Println(runner.FormatArgs(binaryPath, rc))
				}
				return nil
			}

			// Serialise concurrent `up <name>` against the same model. Two
			// simultaneous callers could both Get "not running" and both spawn
			// a process; the lock ensures only one proceeds.
			release, err := appCtx.StateStore.LockModel(name)
			if err != nil {
				if errors.Is(err, state.ErrLockHeld) {
					return fmt.Errorf("%s is already being started by another llamaconfig process", name)
				}
				return err
			}
			defer release()

			// Re-check state under the lock — another process may have just
			// finished starting the model between our first Get and the lock.
			if current, _ := appCtx.StateStore.Get(name); current != nil && current.Status == "running" {
				r := runner.New()
				if r.IsAlive(current) {
					p.Success("%s is already running at http://%s:%d (PID %d)", name, current.Host, current.Port, current.PID)
					return nil
				}
			}

			// Check binary exists before trying to start
			if _, err := os.Stat(binaryPath); err != nil {
				return fmt.Errorf("%s not found at %q — run: llamaconfig install %s",
					filepath.Base(binaryPath), binaryPath, cfg.Backend)
			}

			// Download main model + any configured draft / mmproj artifacts
			// that are not already cached.
			for _, art := range neededArtifacts(cfg, rc) {
				if _, statErr := os.Stat(art.destPath); !os.IsNotExist(statErr) {
					continue
				}
				if flagNoDownload {
					hint := "download the model or remove --no-download"
					if art.source == "huggingface" && art.repo != "" {
						hint = fmt.Sprintf("run: llamaconfig pull %s", art.repo)
					}
					return fmt.Errorf("%s file not found at %q (--no-download is set) — %s", art.kind, art.destPath, hint)
				}
				if art.source != "huggingface" && art.source != "url" {
					return fmt.Errorf("%s file not found at %q", art.kind, art.destPath)
				}
				if err := downloadArtifact(cmd.Context(), cfg, art, p); err != nil {
					return err
				}
			}

			// Interactive mode: run llama-cli in foreground, no state/health needed
			if cfg.Mode == "interactive" {
				p.Info("starting %s in interactive mode (profile: %s)", name, rc.ProfileName)
				r := runner.NewInteractive()
				_, err := r.Start(cmd.Context(), rc)
				return err
			}

			p.Info("starting %s (profile: %s, port: %d)", name, rc.ProfileName, cfg.Server.Port)

			r := runner.New()
			ms, err := r.Start(context.Background(), rc)
			if err != nil {
				return fmt.Errorf("failed to start %s: %w", name, err)
			}

			// Save state before waiting for health (so ps shows it)
			if err := appCtx.StateStore.Put(ms); err != nil {
				p.Warn("could not save state: %v", err)
			}

			p.Info("waiting for %s to become ready...", name)
			if err := runner.WaitHealthy(cmd.Context(), cfg.Server.Host, cfg.Server.Port, cfg.Backend); err != nil {
				ms.Status = "error"
				if putErr := appCtx.StateStore.Put(ms); putErr != nil {
					p.Warn("could not save error state: %v", putErr)
				}
				return fmt.Errorf("%s failed to start: %w", name, err)
			}

			p.Success("%s is ready at http://%s:%d", name, cfg.Server.Host, cfg.Server.Port)
			return nil
		},
	}

	cmd.Flags().IntVar(&flagPort, "port", 0, "override config port")
	cmd.Flags().StringVar(&flagProfile, "profile", "", "force hardware profile (apple_silicon | nvidia | amd | cpu)")
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "print llama.cpp command without running")
	cmd.Flags().BoolVar(&flagNoDownload, "no-download", false, "fail if model is not cached")
	return cmd
}

func resolveBackendBinary(backend string) (string, error) {
	switch backend {
	case "sd":
		return stablediffusioncpp.FindServer()
	case "whisper":
		return whispercpp.FindServer()
	default:
		return llamacpp.FindServer()
	}
}

func profileOverride(name string) *hardware.DetectionResult {
	switch name {
	case "apple_silicon":
		return &hardware.DetectionResult{Class: hardware.ClassAppleSilicon}
	case "nvidia":
		return &hardware.DetectionResult{Class: hardware.ClassNVIDIA}
	case "amd":
		return &hardware.DetectionResult{Class: hardware.ClassAMD}
	case "intel_gpu":
		return &hardware.DetectionResult{Class: hardware.ClassIntelGPU}
	default:
		return &hardware.DetectionResult{Class: hardware.ClassCPU}
	}
}

func placeholderBinaryName(backend string) string {
	switch backend {
	case "sd":
		return "sd-server"
	case "whisper":
		return "whisper-server"
	default:
		return "llama-server"
	}
}

// artifact describes one downloadable file (main model, draft, or mmproj).
type artifact struct {
	kind     string // "model" | "draft" | "mmproj"
	source   string // huggingface | url | local | ""
	repo     string
	file     string
	url      string
	destPath string
	checksum string
}

// neededArtifacts returns every resolvable artifact for the run. Callers
// still must stat each destPath to decide whether to fetch it.
func neededArtifacts(cfg *config.Config, rc *config.RunConfig) []artifact {
	var out []artifact
	if rc.ModelPath != "" {
		out = append(out, artifact{
			kind:     "model",
			source:   cfg.Model.Source,
			repo:     cfg.Model.Repo,
			file:     cfg.Model.File,
			url:      cfg.Model.URL,
			destPath: rc.ModelPath,
			checksum: cfg.Model.Checksum,
		})
	}
	if d := cfg.Model.Draft; d != nil && rc.DraftModelPath != "" {
		out = append(out, artifact{
			kind:     "draft",
			source:   d.Source,
			repo:     d.Repo,
			file:     d.File,
			destPath: rc.DraftModelPath,
		})
	}
	if m := cfg.Model.MMProj; m != nil && rc.MMProjPath != "" {
		out = append(out, artifact{
			kind:     "mmproj",
			source:   m.Source,
			repo:     m.Repo,
			file:     m.File,
			destPath: rc.MMProjPath,
		})
	}
	for _, d := range rc.ExtraDownloads {
		out = append(out, artifact{
			kind:     d.Kind,
			source:   "huggingface",
			repo:     d.Repo,
			file:     d.File,
			destPath: d.DestPath,
		})
	}
	return out
}

func downloadArtifact(ctx context.Context, cfg *config.Config, art artifact, p interface{ Info(string, ...any) }) error {
	p.Info("%s not cached — downloading %s...", art.kind, art.file)

	cacheDir := dirs.ExpandHome(cfg.Model.Download.CacheDir)
	if cacheDir == "" {
		cacheDir = dirs.CacheDir()
	}

	req := &downloader.Request{
		Repo:           art.repo,
		File:           art.file,
		URL:            art.url,
		Token:          resolveToken(""),
		CacheDir:       cacheDir,
		Resume:         *cfg.Model.Download.Resume,
		Connections:    cfg.Model.Download.Connections,
		Checksum:       art.checksum,
		VerifyChecksum: *cfg.Model.Download.VerifyChecksum && art.checksum != "",
	}

	return runDownloadWithProgress(ctx, req, art.file)
}
