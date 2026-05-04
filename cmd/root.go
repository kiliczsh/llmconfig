package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/config"
	"github.com/kiliczsh/llmconfig/internal/dirs"
	"github.com/kiliczsh/llmconfig/internal/output"
	"github.com/kiliczsh/llmconfig/internal/state"
	"github.com/kiliczsh/llmconfig/pkg/llamacpp"
	"github.com/spf13/cobra"
)

type AppContext struct {
	ConfigDir  string
	ModelsDir  string
	LogDir     string
	LlamaBin   string
	Verbose    bool
	NoColor    bool
	Printer    *output.Printer
	StateStore *state.Store
}

type contextKey struct{}

func appCtxFrom(ctx context.Context) *AppContext {
	v, _ := ctx.Value(contextKey{}).(*AppContext)
	return v
}

var (
	flagConfigDir string
	flagLlamaBin  string
	flagNoColor   bool
	flagVerbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "llmconfig",
	Short: "Local Large Model Config — manage llama.cpp, stable-diffusion.cpp, and whisper.cpp",
	Long: `llmconfig — Local Large Model Config.

Manage local inference with llama.cpp, stable-diffusion.cpp, and
whisper.cpp. Define your model once in a YAML file; llmconfig handles
downloading, starting, stopping, and monitoring.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	// Match Use to how the user invoked us, so `llmc --help` shows
	// "Usage: llmc [command]" instead of the hardcoded project name.
	if len(os.Args) > 0 {
		invoked := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		if invoked != "" {
			rootCmd.Use = invoked
		}
	}
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagConfigDir, "config-dir", "", "override config directory (default: ~/.llmconfig)")
	rootCmd.PersistentFlags().StringVar(&flagLlamaBin, "llama-bin", "", "override llama.cpp binary path")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")

	cobra.OnInitialize(func() {
		if flagConfigDir != "" {
			os.Setenv("LLMCONFIG_CONFIG_DIR", flagConfigDir)
		}
	})

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := dirs.EnsureAll(); err != nil {
			return fmt.Errorf("failed to create llmconfig directories: %w", err)
		}

		store := state.NewStore()
		if err := store.EnsureDir(); err != nil {
			return err
		}

		configDir := flagConfigDir
		if configDir == "" {
			configDir = dirs.ConfigDir()
		}

		llamaBin := flagLlamaBin
		if llamaBin == "" {
			llamaBin = findLlamaBinary()
		}

		appCtx := &AppContext{
			ConfigDir:  configDir,
			ModelsDir:  dirs.ModelsDir(),
			LogDir:     dirs.LogDir(),
			LlamaBin:   llamaBin,
			Verbose:    flagVerbose,
			NoColor:    flagNoColor,
			Printer:    output.New(flagNoColor),
			StateStore: store,
		}

		// One-shot migration of legacy *.yaml configs in the user's
		// configs dir to *.llmc. Marker file gates the run so we only
		// scan once per machine; subsequent invocations short-circuit.
		runYamlMigration(configDir, appCtx.Printer)

		ctx := context.WithValue(cmd.Context(), contextKey{}, appCtx)
		cmd.SetContext(ctx)
		return nil
	}

	rootCmd.AddCommand(
		newUpCmd(),
		newDownCmd(),
		newRestartCmd(),
		newStateCmd(),
		newPsCmd(),
		newLogsCmd(),
		newStatsCmd(),
		newStatusCmd(),
		newPullCmd(),
		newAddCmd(),
		newModelsCmd(),
		newRmCmd(),
		newInitCmd(),
		newConfigCmd(),
		newValidateCmd(),
		newInspectCmd(),
		newHardwareCmd(),
		newInstallCmd(),
		newLlamaCmd(),
		newIkLlamaCmd(),
		newSdCmd(),
		newWhisperCmd(),
		newFilesCmd(),
		newVersionCmd(),
		newBenchCmd(),
		newCompatCmd(),
		newArchiveCmd(),
		newImportCmd(),
		newGatewayCmd(),
		newUpdateCmd(),
	)
}

func findLlamaBinary() string {
	if path, err := llamacpp.FindServer(); err == nil {
		return path
	}
	return "llama-server"
}

// yamlMigrationMarker is the dirs.MarkerFile name that records a
// completed .yaml→.llmc migration sweep. The presence of the file is
// the only signal — its contents are ignored.
const yamlMigrationMarker = "yaml-to-llmc-v1"

// runYamlMigration renames any leftover .yaml configs in configDir to
// .llmc on first run, then writes a marker file so future invocations
// skip the work. Failures are logged but never abort the user's
// command — a botched migration must not block `up`/`models`/etc.
func runYamlMigration(configDir string, p *output.Printer) {
	marker := dirs.MarkerFile(yamlMigrationMarker)
	if _, err := os.Stat(marker); err == nil {
		return
	}

	if _, err := os.Stat(configDir); err != nil {
		// No config dir yet (fresh install) — nothing to migrate.
		// Drop the marker so we don't keep checking.
		_ = writeMarker(marker)
		return
	}

	res, err := config.MigrateLegacyConfigs(configDir)
	if err != nil {
		p.Warn("config migration: %v (you can rename %s/*.yaml to *.llmc by hand)", err, configDir)
		// Don't write the marker — we want to retry next run.
		return
	}

	if len(res.Renamed) > 0 {
		p.Info("migrated %d yaml config(s) to .llmc (backups: *.yaml.bak in %s)",
			len(res.Renamed), configDir)
	}
	if len(res.Skipped) > 0 {
		p.Warn("skipped %d yaml config(s) in %s — both .llmc and .yaml exist; remove one to disambiguate",
			len(res.Skipped), configDir)
	}

	_ = writeMarker(marker)
}

func writeMarker(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, nil, 0644)
}
