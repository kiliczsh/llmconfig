package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kiliczsh/llamaconfig/internal/dirs"
	"github.com/kiliczsh/llamaconfig/internal/output"
	"github.com/kiliczsh/llamaconfig/internal/state"
	"github.com/kiliczsh/llamaconfig/pkg/llamacpp"
	"github.com/spf13/cobra"
)

type AppContext struct {
	ConfigDir  string
	CacheDir   string
	LogDir     string
	LlamaBin   string
	Verbose    bool
	NoColor    bool
	JSONOutput bool
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
	flagJSON      bool
)

var rootCmd = &cobra.Command{
	Use:   "llamaconfig",
	Short: "Config-driven CLI for managing llama.cpp model inference",
	Long: `llamaconfig — manage local LLM inference with llama.cpp.

Define your model once in a YAML file. llamaconfig handles
downloading, starting, stopping, and monitoring.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagConfigDir, "config-dir", "", "override config directory (default: ~/.llamaconfig)")
	rootCmd.PersistentFlags().StringVar(&flagLlamaBin, "llama-bin", "", "override llama.cpp binary path")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "JSON output (where supported)")

	cobra.OnInitialize(func() {
		if flagConfigDir != "" {
			os.Setenv("LLAMACONFIG_CONFIG_DIR", flagConfigDir)
		}
	})

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := dirs.EnsureAll(); err != nil {
			return fmt.Errorf("failed to create llamaconfig directories: %w", err)
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
			CacheDir:   dirs.CacheDir(),
			LogDir:     dirs.LogDir(),
			LlamaBin:   llamaBin,
			Verbose:    flagVerbose,
			NoColor:    flagNoColor,
			JSONOutput: flagJSON,
			Printer:    output.New(flagNoColor, flagJSON),
			StateStore: store,
		}

		ctx := context.WithValue(cmd.Context(), contextKey{}, appCtx)
		cmd.SetContext(ctx)
		return nil
	}

	rootCmd.AddCommand(
		newUpCmd(),
		newDownCmd(),
		newRestartCmd(),
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
		newLlamaCmd(),
		newCacheCmd(),
		newVersionCmd(),
	)
}

func findLlamaBinary() string {
	if path, err := llamacpp.FindServer(); err == nil {
		return path
	}
	return "llama-server"
}
