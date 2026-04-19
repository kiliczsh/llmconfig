package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/state"
)

type interactiveRunner struct{}

// NewInteractive returns a runner that launches llama-cli in the foreground.
func NewInteractive() Runner {
	return &interactiveRunner{}
}

func (r *interactiveRunner) Start(ctx context.Context, rc *config.RunConfig) (*state.ModelState, error) {
	args := buildInteractiveArgs(rc)

	// Derive llama-cli binary path from llama-server path
	binaryPath := deriveCLIBinary(rc.BinaryPath)

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Interactive runs in foreground — block until it exits
	if err := cmd.Run(); err != nil {
		// Exit code 1 from llama-cli on normal quit is fine
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("llama-cli: %w", err)
	}
	return nil, nil
}

func (r *interactiveRunner) Stop(_ context.Context, _ *state.ModelState, _ int) error {
	return nil // interactive process blocks; ctrl-c handled by OS
}

func (r *interactiveRunner) IsAlive(_ *state.ModelState) bool {
	return false
}

func buildInteractiveArgs(rc *config.RunConfig) []string {
	cfg := rc.Config
	p := rc.Profile
	var args []string

	add := func(flag string, val string) { args = append(args, flag, val) }
	addIf := func(flag string, cond bool) {
		if cond {
			args = append(args, flag)
		}
	}

	add("--model", rc.ModelPath)
	add("-ngl", strconv.Itoa(p.NGPULayers))

	// Skip --threads on GPU backends — CUDA/Metal handle parallelism internally
	if p.Threads > 0 && !p.CUDA && !p.Metal && !p.ROCm {
		add("--threads", strconv.Itoa(p.Threads))
	}

	add("--ctx-size", strconv.Itoa(cfg.Context.NCtx))
	add("--batch-size", strconv.Itoa(cfg.Context.NBatch))
	if cfg.Context.FlashAttention {
		add("--flash-attn", "on")
	}
	addIf("--no-mmap", !cfg.Context.MMap)
	addIf("--mlock", cfg.Context.MLock)

	// --conversation enables chat mode; Enter submits (no multiline — that requires \ to submit)
	args = append(args, "--conversation")

	// Do not pass --chat-template in conversation mode — the model's embedded
	// metadata template takes precedence and explicit overrides break formatting.
	if cfg.Chat.SystemPrompt != "" {
		add("--system-prompt", cfg.Chat.SystemPrompt)
	}

	// Sampling
	s := cfg.Sampling
	add("--temp", fmt.Sprintf("%.4f", s.Temperature))
	add("--top-k", strconv.Itoa(s.TopK))
	add("--top-p", fmt.Sprintf("%.4f", s.TopP))

	if rc.MMProjPath != "" {
		add("--mmproj", rc.MMProjPath)
	}

	return args
}

// FormatInteractiveArgs returns the llama-cli command as a human-readable string.
func FormatInteractiveArgs(cliBin string, rc *config.RunConfig) string {
	args := buildInteractiveArgs(rc)
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, cliBin)
	for _, a := range args {
		if containsSpace(a) {
			parts = append(parts, fmt.Sprintf("%q", a))
		} else {
			parts = append(parts, a)
		}
	}
	return joinArgs(parts)
}

func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			return true
		}
	}
	return false
}

func joinArgs(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

// DeriveCLIBinary is the exported version for use in cmd/.
func DeriveCLIBinary(serverBin string) string {
	return deriveCLIBinary(serverBin)
}

// deriveCLIBinary replaces "llama-server" with "llama-cli" in the binary path.
func deriveCLIBinary(serverBin string) string {
	if serverBin == "" || serverBin == "llama-server" {
		if path, err := exec.LookPath("llama-cli"); err == nil {
			return path
		}
		return "llama-cli"
	}
	// Replace last segment
	for i := len(serverBin) - 1; i >= 0; i-- {
		if serverBin[i] == '/' || serverBin[i] == '\\' {
			dir := serverBin[:i+1]
			name := serverBin[i+1:]
			if name == "llama-server" || name == "llama-server.exe" {
				cli := dir + "llama-cli"
				if name == "llama-server.exe" {
					cli += ".exe"
				}
				return cli
			}
			break
		}
	}
	if path, err := exec.LookPath("llama-cli"); err == nil {
		return path
	}
	return "llama-cli"
}
