package runner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kiliczsh/llamaconfig/internal/config"
)

// BuildArgs converts a RunConfig into the argv slice for the appropriate backend binary.
// This is a pure function — both --dry-run and inspect use it.
func BuildArgs(rc *config.RunConfig) []string {
	switch rc.Backend {
	case "whisper":
		return buildWhisperArgs(rc)
	case "sd":
		return buildSDArgs(rc)
	default:
		return buildLlamaArgs(rc)
	}
}

func buildLlamaArgs(rc *config.RunConfig) []string {
	cfg := rc.Config
	p := rc.Profile
	var args []string

	add := func(flag string, value ...string) {
		args = append(args, flag)
		args = append(args, value...)
	}
	addIf := func(flag string, cond bool) {
		if cond {
			args = append(args, flag)
		}
	}

	// Model
	add("--model", rc.ModelPath)

	// Server
	add("--host", cfg.Server.Host)
	add("--port", strconv.Itoa(cfg.Server.Port))

	if cfg.Server.APIKey != "" {
		add("--api-key", cfg.Server.APIKey)
	}
	if cfg.Server.Parallel > 1 {
		add("--parallel", strconv.Itoa(cfg.Server.Parallel))
	}
	for _, origin := range cfg.Server.CORSOrigins {
		add("--cors", origin)
	}

	// Endpoints
	addIf("--metrics", cfg.Server.Endpoints.Metrics)
	addIf("--no-slots", !*cfg.Server.Endpoints.Slots)
	addIf("--embedding", cfg.Server.Endpoints.Embeddings)

	// Hardware / GPU
	add("-ngl", strconv.Itoa(p.NGPULayers))

	if len(p.Devices) > 0 {
		add("--device", strings.Join(p.Devices, ","))
	}
	if len(p.TensorSplit) > 0 {
		parts := make([]string, len(p.TensorSplit))
		for i, v := range p.TensorSplit {
			parts[i] = fmt.Sprintf("%.2f", v)
		}
		add("--tensor-split", strings.Join(parts, ","))
	}

	// Threads
	if p.Threads > 0 {
		add("--threads", strconv.Itoa(p.Threads))
	}
	if p.ThreadsBatch > 0 {
		add("--threads-batch", strconv.Itoa(p.ThreadsBatch))
	}

	// Context
	ctx := cfg.Context
	add("--ctx-size", strconv.Itoa(ctx.NCtx))
	add("--batch-size", strconv.Itoa(ctx.NBatch))
	if ctx.NUBatch > 0 && ctx.NUBatch != ctx.NBatch {
		add("--ubatch-size", strconv.Itoa(ctx.NUBatch))
	}
	if ctx.NKeep > 0 {
		add("--keep", strconv.Itoa(ctx.NKeep))
	}
	if ctx.CacheTypeK != "" && ctx.CacheTypeK != "f16" {
		add("--cache-type-k", ctx.CacheTypeK)
	}
	if ctx.CacheTypeV != "" && ctx.CacheTypeV != "f16" {
		add("--cache-type-v", ctx.CacheTypeV)
	}
	addIf("--no-mmap", !*ctx.MMap)
	addIf("--mlock", ctx.MLock)
	if ctx.FlashAttention {
		add("--flash-attn", "on")
	}
	if ctx.NCPUMoE > 0 {
		add("--cpu-moe-layers", strconv.Itoa(ctx.NCPUMoE))
	}

	// Sampling
	s := cfg.Sampling
	add("--temp", fmt.Sprintf("%.4f", s.Temperature))
	add("--top-k", strconv.Itoa(s.TopK))
	add("--top-p", fmt.Sprintf("%.4f", s.TopP))
	add("--min-p", fmt.Sprintf("%.4f", s.MinP))
	if s.RepeatPenalty != 1.0 {
		add("--repeat-penalty", fmt.Sprintf("%.4f", s.RepeatPenalty))
		add("--repeat-last-n", strconv.Itoa(s.RepeatLastN))
	}
	if s.DryMultiplier > 0 {
		add("--dry-multiplier", fmt.Sprintf("%.4f", s.DryMultiplier))
		add("--dry-base", fmt.Sprintf("%.4f", s.DryBase))
		add("--dry-allowed-length", strconv.Itoa(s.DryAllowedLength))
		add("--dry-penalty-last-n", strconv.Itoa(s.DryPenaltyLastN))
	}
	if s.Mirostat > 0 {
		add("--mirostat", strconv.Itoa(s.Mirostat))
		add("--mirostat-tau", fmt.Sprintf("%.4f", s.MirostatTau))
		add("--mirostat-eta", fmt.Sprintf("%.4f", s.MirostatEta))
	}
	if s.Samplers != "" {
		add("--samplers", s.Samplers)
	}

	// Chat template
	if cfg.Chat.Template != "" {
		add("--chat-template", cfg.Chat.Template)
	}
	if cfg.Chat.SystemPrompt != "" {
		add("--system-prompt", cfg.Chat.SystemPrompt)
	}
	if cfg.Chat.Jinja {
		args = append(args, "--jinja")
	}

	// RoPE
	rope := cfg.Rope
	if rope.Scaling != "" {
		add("--rope-scaling", rope.Scaling)
	}
	if rope.FreqBase > 0 {
		add("--rope-freq-base", fmt.Sprintf("%.1f", rope.FreqBase))
	}
	if rope.FreqScale > 0 && rope.FreqScale != 1.0 {
		add("--rope-freq-scale", fmt.Sprintf("%.6f", rope.FreqScale))
	}
	if rope.Scaling == "yarn" {
		add("--yarn-ext-factor", fmt.Sprintf("%.4f", rope.YarnExtFactor))
		add("--yarn-attn-factor", fmt.Sprintf("%.4f", rope.YarnAttnFactor))
		if rope.YarnOrigCtx > 0 {
			add("--yarn-orig-ctx", strconv.Itoa(rope.YarnOrigCtx))
		}
	}

	// Draft model (speculative decoding)
	if rc.DraftModelPath != "" {
		add("--model-draft", rc.DraftModelPath)
		if cfg.Model.Draft != nil && cfg.Model.Draft.DraftN > 0 {
			add("--draft", strconv.Itoa(cfg.Model.Draft.DraftN))
		}
	}

	// Multimodal projection
	if rc.MMProjPath != "" {
		add("--mmproj", rc.MMProjPath)
	}

	// CPU-specific
	if p.CPURange != "" {
		add("--cpu-range", p.CPURange)
	}
	if p.CPUStrict {
		args = append(args, "--cpu-strict", "1")
	}
	if p.NUMA != "" {
		add("--numa", p.NUMA)
	}

	return args
}

// FormatArgs returns the BuildArgs result as a human-readable command string.
func FormatArgs(binaryPath string, rc *config.RunConfig) string {
	args := BuildArgs(rc)
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, binaryPath)
	for _, a := range args {
		if strings.ContainsAny(a, " \t\n") {
			parts = append(parts, fmt.Sprintf("%q", a))
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}
