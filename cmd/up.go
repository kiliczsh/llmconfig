package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/dirs"
	"github.com/kiliczsh/llamaconfig/internal/downloader"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
	"github.com/kiliczsh/llamaconfig/internal/runner"
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

			// Select binary based on backend
			binaryPath, err := resolveBackendBinary(cfg.Backend)
			if err != nil {
				return err
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

			// Check binary exists before trying to start
			if _, err := os.Stat(binaryPath); err != nil {
				return fmt.Errorf("%s not found at %q — run: llamaconfig install %s",
					filepath.Base(binaryPath), binaryPath, cfg.Backend)
			}

			// Download model if needed
			if rc.ModelPath != "" {
				if _, statErr := os.Stat(rc.ModelPath); os.IsNotExist(statErr) {
					if flagNoDownload {
						return fmt.Errorf("model file not found at %s (--no-download is set)", rc.ModelPath)
					}
					if err := downloadModel(cmd.Context(), cfg, rc, p); err != nil {
						return err
					}
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

func downloadModel(ctx context.Context, cfg *config.Config, rc *config.RunConfig, p interface{ Info(string, ...any) }) error {
	if cfg.Model.Source != "huggingface" && cfg.Model.Source != "url" {
		return fmt.Errorf("model file not found at %s", rc.ModelPath)
	}

	p.Info("model not cached — downloading %s...", cfg.Model.File)

	token := resolveToken("")
	cacheDir := dirs.ExpandHome(cfg.Model.Download.CacheDir)
	if cacheDir == "" {
		cacheDir = dirs.CacheDir()
	}

	req := &downloader.Request{
		Repo:           cfg.Model.Repo,
		File:           cfg.Model.File,
		URL:            cfg.Model.URL,
		Token:          token,
		CacheDir:       cacheDir,
		Resume:         *cfg.Model.Download.Resume,
		Connections:    cfg.Model.Download.Connections,
		Checksum:       cfg.Model.Checksum,
		VerifyChecksum: *cfg.Model.Download.VerifyChecksum,
	}

	return runDownloadWithProgress(ctx, req, cfg.Model.File)
}
