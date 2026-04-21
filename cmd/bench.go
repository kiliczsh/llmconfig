package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kiliczsh/llamaconfig/internal/bench"
	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/dirs"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
	"github.com/kiliczsh/llamaconfig/internal/runner"
	"github.com/spf13/cobra"
)

var reTimingLine = regexp.MustCompile(`\[ Prompt: ([\d.]+) t/s \| Generation: ([\d.]+) t/s \]`)

const benchPrompt = "Explain the difference between a compiler and an interpreter in simple terms."

func newBenchCmd() *cobra.Command {
	var flagRuns int
	var flagTokens int

	cmd := &cobra.Command{
		Use:   "bench <name>",
		Short: "Benchmark a model's inference speed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			name := args[0]

			cfg, err := config.Load(name, appCtx.ConfigDir)
			if err != nil {
				return err
			}
			config.ApplyDefaults(cfg)
			if err := config.Validate(cfg); err != nil {
				return err
			}

			hw := hardware.Detect()
			rc, err := config.Resolve(cfg, hw, appCtx.LlamaBin)
			if err != nil {
				return err
			}

			if _, err := os.Stat(appCtx.LlamaBin); err != nil {
				return fmt.Errorf("llama-server binary not found at %q — run: llamaconfig llama --install", appCtx.LlamaBin)
			}

			if pids := runningLlamaPIDs(); len(pids) > 0 {
				return fmt.Errorf("llama-cli/llama-server already running (PIDs: %v) — stop them first with: llamaconfig down", pids)
			}

			if _, statErr := os.Stat(rc.ModelPath); os.IsNotExist(statErr) {
				if err := downloadModel(cmd.Context(), cfg, rc, p); err != nil {
					return err
				}
			}

			cliBin := runner.DeriveCLIBinary(appCtx.LlamaBin, "llama")
			baseArgs := buildBenchArgs(rc, flagTokens)

			p.Info("benchmarking %s (%d runs)...", name, flagRuns)
			fmt.Println()

			var runs []bench.Run
			for i := 1; i <= flagRuns; i++ {
				promptTPS, genTPS, err := runBenchOnce(cliBin, baseArgs)
				if err != nil {
					return fmt.Errorf("run %d failed: %w", i, err)
				}
				runs = append(runs, bench.Run{
					PromptTPS:   promptTPS,
					GenerateTPS: genTPS,
					Timestamp:   time.Now(),
				})
				fmt.Printf("  run %-2d  prompt: %.1f t/s  generation: %.1f t/s\n", i, promptTPS, genTPS)
			}

			avgPrompt, avgGen := bench.Avg(runs)
			fmt.Printf("\n  avg     prompt: %.1f t/s  generation: %.1f t/s\n\n", avgPrompt, avgGen)

			result := &bench.Result{
				Name:           name,
				Runs:           runs,
				AvgPromptTPS:   avgPrompt,
				AvgGenerateTPS: avgGen,
			}
			if err := bench.Save(dirs.BenchDir(), result); err != nil {
				p.Warn("could not save bench result: %v", err)
			} else {
				p.Success("saved to %s/%s.json", dirs.BenchDir(), name)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&flagRuns, "runs", 3, "number of benchmark runs")
	cmd.Flags().IntVar(&flagTokens, "tokens", 100, "tokens to generate per run")
	return cmd
}

func buildBenchArgs(rc *config.RunConfig, nTokens int) []string {
	cfg := rc.Config
	p := rc.Profile
	var args []string

	add := func(flag, val string) { args = append(args, flag, val) }

	add("--model", rc.ModelPath)
	add("-ngl", strconv.Itoa(p.NGPULayers))
	add("--ctx-size", strconv.Itoa(cfg.Context.NCtx))
	add("--batch-size", strconv.Itoa(cfg.Context.NBatch))
	add("-p", benchPrompt)
	add("-n", strconv.Itoa(nTokens))

	return args
}

func runBenchOnce(cliBin string, args []string) (promptTPS, genTPS float64, err error) {
	cmd := exec.Command(cliBin, args...)
	cmd.Stdin = nil
	cmd.Stderr = io.Discard

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, 0, err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		p, g, ok := parseTimingLine(scanner.Text())
		if ok {
			promptTPS, genTPS = p, g
			break
		}
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	return promptTPS, genTPS, nil
}

func runningLlamaPIDs() []string {
	out, err := exec.Command("pgrep", "-f", "llama-cli").Output()
	if err != nil {
		return nil
	}
	var pids []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		if pid := strings.TrimSpace(scanner.Text()); pid != "" {
			pids = append(pids, pid)
		}
	}
	return pids
}

func parseTimingLine(line string) (promptTPS, genTPS float64, ok bool) {
	m := reTimingLine.FindStringSubmatch(line)
	if m == nil {
		return 0, 0, false
	}
	promptTPS, _ = strconv.ParseFloat(m[1], 64)
	genTPS, _ = strconv.ParseFloat(m[2], 64)
	return promptTPS, genTPS, true
}
