package runner

import (
	"encoding/json"
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
	if p.SplitMode != "" {
		add("--split-mode", p.SplitMode)
	}
	if p.MainGPU >= 0 {
		add("--main-gpu", strconv.Itoa(p.MainGPU))
	}
	if p.Priority != 0 {
		add("--prio", strconv.Itoa(p.Priority))
	}
	if p.Fit != "" {
		add("--fit", p.Fit)
	}
	if len(p.FitTarget) > 0 {
		parts := make([]string, len(p.FitTarget))
		for i, v := range p.FitTarget {
			parts[i] = strconv.Itoa(v)
		}
		add("--fit-target", strings.Join(parts, ","))
	}
	if p.FitCtx > 0 {
		add("--fit-ctx", strconv.Itoa(p.FitCtx))
	}
	for _, ot := range p.OverrideTensor {
		add("--override-tensor", ot)
	}
	addIf("--cpu-moe", p.CPUMoE)
	if p.CPUMask != "" {
		add("--cpu-mask", p.CPUMask)
	}
	if p.CPUMaskBatch != "" {
		add("--cpu-mask-batch", p.CPUMaskBatch)
	}
	if p.Poll != nil {
		add("--poll", strconv.Itoa(*p.Poll))
	}
	if p.PollBatch != nil {
		add("--poll-batch", strconv.Itoa(*p.PollBatch))
	}
	if p.PrioBatch != 0 {
		add("--prio-batch", strconv.Itoa(p.PrioBatch))
	}
	if p.Repack != nil && !*p.Repack {
		args = append(args, "--no-repack")
	}
	if p.NoHost {
		args = append(args, "--no-host")
	}
	if p.OpOffload != nil && !*p.OpOffload {
		args = append(args, "--no-op-offload")
	}
	if p.RPC != "" {
		add("--rpc", p.RPC)
	}
	if p.DirectIO != nil && *p.DirectIO {
		args = append(args, "--direct-io")
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
	if ctx.NPredict != 0 {
		add("--predict", strconv.Itoa(ctx.NPredict))
	}
	addIf("--context-shift", ctx.ContextShift)
	if ctx.KVOffload != nil && !*ctx.KVOffload {
		args = append(args, "--no-kv-offload")
	}
	addIf("--swa-full", ctx.SWAFull)
	if ctx.CacheRAM != 0 {
		add("--cache-ram", strconv.Itoa(ctx.CacheRAM))
	}
	if ctx.ImageMinTokens > 0 {
		add("--image-min-tokens", strconv.Itoa(ctx.ImageMinTokens))
	}
	if ctx.ImageMaxTokens > 0 {
		add("--image-max-tokens", strconv.Itoa(ctx.ImageMaxTokens))
	}
	addIf("--check-tensors", ctx.CheckTensors)
	if ctx.CtxCheckpoints > 0 {
		add("--ctx-checkpoints", strconv.Itoa(ctx.CtxCheckpoints))
	}
	if ctx.CheckpointEveryNTokens != 0 {
		add("--checkpoint-every-n-tokens", strconv.Itoa(ctx.CheckpointEveryNTokens))
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
	if s.PresencePenalty != 0 {
		add("--presence-penalty", fmt.Sprintf("%.4f", s.PresencePenalty))
	}
	if s.FrequencyPenalty != 0 {
		add("--frequency-penalty", fmt.Sprintf("%.4f", s.FrequencyPenalty))
	}
	if s.DryMultiplier > 0 {
		add("--dry-multiplier", fmt.Sprintf("%.4f", s.DryMultiplier))
		add("--dry-base", fmt.Sprintf("%.4f", s.DryBase))
		add("--dry-allowed-length", strconv.Itoa(s.DryAllowedLength))
		add("--dry-penalty-last-n", strconv.Itoa(s.DryPenaltyLastN))
	}
	if s.DynatempRange != 0 {
		add("--dynatemp-range", fmt.Sprintf("%.4f", s.DynatempRange))
		add("--dynatemp-exp", fmt.Sprintf("%.4f", s.DynatempExp))
	}
	if s.XTCProbability != 0 {
		add("--xtc-probability", fmt.Sprintf("%.4f", s.XTCProbability))
		add("--xtc-threshold", fmt.Sprintf("%.4f", s.XTCThreshold))
	}
	if s.Typical != 0 {
		add("--typical", fmt.Sprintf("%.4f", s.Typical))
	}
	if s.TopNSigma != 0 {
		add("--top-nsigma", fmt.Sprintf("%.4f", s.TopNSigma))
	}
	if s.AdaptiveTarget != 0 {
		add("--adaptive-target", fmt.Sprintf("%.4f", s.AdaptiveTarget))
		if s.AdaptiveDecay != 0 {
			add("--adaptive-decay", fmt.Sprintf("%.4f", s.AdaptiveDecay))
		}
	}
	for _, breaker := range s.DrySequenceBreakers {
		add("--dry-sequence-breaker", breaker)
	}
	if s.Mirostat > 0 {
		add("--mirostat", strconv.Itoa(s.Mirostat))
		add("--mirostat-tau", fmt.Sprintf("%.4f", s.MirostatTau))
		add("--mirostat-eta", fmt.Sprintf("%.4f", s.MirostatEta))
	}
	if s.Samplers != "" {
		add("--samplers", s.Samplers)
	}
	if s.Seed != 0 {
		add("--seed", strconv.FormatInt(s.Seed, 10))
	}
	if s.Grammar != "" {
		add("--grammar", s.Grammar)
	}
	if s.GrammarFile != "" {
		add("--grammar-file", s.GrammarFile)
	}
	if s.JSONSchema != "" {
		add("--json-schema", s.JSONSchema)
	}
	if s.JSONSchemaFile != "" {
		add("--json-schema-file", s.JSONSchemaFile)
	}
	addIf("--backend-sampling", s.BackendSampling)

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
	if len(cfg.Chat.TemplateKwargs) > 0 {
		if b, err := json.Marshal(cfg.Chat.TemplateKwargs); err == nil {
			add("--chat-template-kwargs", string(b))
		}
	}
	if cfg.Chat.Reasoning != "" {
		add("--reasoning", cfg.Chat.Reasoning)
	}
	if cfg.Chat.ReasoningBudget != nil {
		add("--reasoning-budget", strconv.Itoa(*cfg.Chat.ReasoningBudget))
	}
	if cfg.Chat.ReasoningFormat != "" {
		add("--reasoning-format", cfg.Chat.ReasoningFormat)
	}
	if cfg.Chat.TemplateFile != "" {
		add("--chat-template-file", cfg.Chat.TemplateFile)
	}
	addIf("--skip-chat-parsing", cfg.Chat.SkipChatParsing)

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
		if d := cfg.Model.Draft; d != nil {
			if d.DraftN > 0 {
				add("--draft", strconv.Itoa(d.DraftN))
			}
			if d.DraftMin > 0 {
				add("--draft-min", strconv.Itoa(d.DraftMin))
			}
			if d.DraftPMin > 0 {
				add("--draft-p-min", fmt.Sprintf("%.4f", d.DraftPMin))
			}
			if d.NCtx > 0 {
				add("--ctx-size-draft", strconv.Itoa(d.NCtx))
			}
			if d.NGPULayers > 0 {
				add("--n-gpu-layers-draft", strconv.Itoa(d.NGPULayers))
			}
			if len(d.Devices) > 0 {
				add("--device-draft", strings.Join(d.Devices, ","))
			}
			if d.CacheTypeK != "" && d.CacheTypeK != "f16" {
				add("--cache-type-k-draft", d.CacheTypeK)
			}
			if d.CacheTypeV != "" && d.CacheTypeV != "f16" {
				add("--cache-type-v-draft", d.CacheTypeV)
			}
			if d.SpecReplaceTarget != "" && d.SpecReplaceDraft != "" {
				add("--spec-replace", d.SpecReplaceTarget, d.SpecReplaceDraft)
			}
		}
	}

	// LoRA
	for _, lora := range cfg.Model.LoRA {
		add("--lora", lora)
	}
	if len(cfg.Model.LoRAScaled) > 0 {
		add("--lora-scaled", strings.Join(cfg.Model.LoRAScaled, ","))
	}

	// Control vectors
	for _, cv := range cfg.Model.ControlVector {
		add("--control-vector", cv)
	}
	if len(cfg.Model.ControlVectorScaled) > 0 {
		add("--control-vector-scaled", strings.Join(cfg.Model.ControlVectorScaled, ","))
	}
	if cfg.Model.ControlVectorLayerStart >= 0 && cfg.Model.ControlVectorLayerEnd >= 0 {
		add("--control-vector-layer-range",
			strconv.Itoa(cfg.Model.ControlVectorLayerStart),
			strconv.Itoa(cfg.Model.ControlVectorLayerEnd))
	}

	// Model metadata overrides
	for _, kv := range cfg.Model.OverrideKV {
		add("--override-kv", kv)
	}

	// Multimodal projection
	if rc.MMProjPath != "" {
		add("--mmproj", rc.MMProjPath)
		if cfg.Model.MMProj != nil && cfg.Model.MMProj.Offload != nil && !*cfg.Model.MMProj.Offload {
			args = append(args, "--no-mmproj-offload")
		}
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

	// Logging
	log := cfg.Logging
	if log.File != "" {
		add("--log-file", log.File)
	}
	if log.Colors != "" && log.Colors != "auto" {
		add("--log-colors", log.Colors)
	}
	addIf("--log-prefix", log.Prefix)
	addIf("--log-timestamps", log.Timestamps)

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
