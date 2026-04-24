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
	if cfg.Server.APIKeyFile != "" {
		add("--api-key-file", cfg.Server.APIKeyFile)
	}
	if cfg.Server.Parallel > 1 {
		add("--parallel", strconv.Itoa(cfg.Server.Parallel))
	}
	for _, origin := range cfg.Server.CORSOrigins {
		add("--cors", origin)
	}
	if cfg.Server.Timeout > 0 {
		add("--timeout", strconv.Itoa(cfg.Server.Timeout))
	}
	if cfg.Server.ThreadsHTTP > 0 {
		add("--threads-http", strconv.Itoa(cfg.Server.ThreadsHTTP))
	}
	addIf("--reuse-port", cfg.Server.ReusePort)
	if cfg.Server.StaticPath != "" {
		add("--path", cfg.Server.StaticPath)
	}
	if cfg.Server.APIPrefix != "" {
		add("--api-prefix", cfg.Server.APIPrefix)
	}
	if cfg.Server.SSLKeyFile != "" {
		add("--ssl-key-file", cfg.Server.SSLKeyFile)
	}
	if cfg.Server.SSLCertFile != "" {
		add("--ssl-cert-file", cfg.Server.SSLCertFile)
	}
	if cfg.Server.CachePrompt != nil && !*cfg.Server.CachePrompt {
		args = append(args, "--no-cache-prompt")
	}
	if cfg.Server.CacheReuse > 0 {
		add("--cache-reuse", strconv.Itoa(cfg.Server.CacheReuse))
	}
	if cfg.Server.SlotPromptSimilarity != 0 {
		add("--slot-prompt-similarity", fmt.Sprintf("%.4f", cfg.Server.SlotPromptSimilarity))
	}
	if cfg.Server.SlotSavePath != "" {
		add("--slot-save-path", cfg.Server.SlotSavePath)
	}
	if cfg.Server.SleepIdleSeconds != 0 {
		add("--sleep-idle-seconds", strconv.Itoa(cfg.Server.SleepIdleSeconds))
	}
	addIf("--lora-init-without-apply", cfg.Server.LoRAInitWithoutApply)
	if cfg.Server.KVUnified != nil && !*cfg.Server.KVUnified {
		args = append(args, "--no-kv-unified")
	}
	if cfg.Server.ClearIdle != nil && !*cfg.Server.ClearIdle {
		args = append(args, "--no-clear-idle")
	}
	if cfg.Server.ContBatching != nil && !*cfg.Server.ContBatching {
		args = append(args, "--no-cont-batching")
	}
	if cfg.Server.Alias != "" {
		add("--alias", cfg.Server.Alias)
	}
	if cfg.Server.Tags != "" {
		add("--tags", cfg.Server.Tags)
	}
	if cfg.Server.WebUI != nil && !*cfg.Server.WebUI {
		args = append(args, "--no-webui")
	}
	if cfg.Server.WebUIConfig != "" {
		add("--webui-config", cfg.Server.WebUIConfig)
	}
	if cfg.Server.WebUIConfigFile != "" {
		add("--webui-config-file", cfg.Server.WebUIConfigFile)
	}
	addIf("--webui-mcp-proxy", cfg.Server.WebUIMCPProxy)
	if cfg.Server.Tools != "" {
		add("--tools", cfg.Server.Tools)
	}
	if cfg.Server.Pooling != "" {
		add("--pooling", cfg.Server.Pooling)
	}
	addIf("--spm-infill", cfg.Server.SPMInfill)
	if cfg.Server.PrefillAssistant != nil && !*cfg.Server.PrefillAssistant {
		args = append(args, "--no-prefill-assistant")
	}
	if cfg.Server.MediaPath != "" {
		add("--media-path", cfg.Server.MediaPath)
	}
	if cfg.Server.ModelsDir != "" {
		add("--models-dir", cfg.Server.ModelsDir)
	}
	if cfg.Server.ModelsPreset != "" {
		add("--models-preset", cfg.Server.ModelsPreset)
	}
	if cfg.Server.ModelsMax >= 0 {
		add("--models-max", strconv.Itoa(cfg.Server.ModelsMax))
	}
	if cfg.Server.ModelsAutoload != nil && !*cfg.Server.ModelsAutoload {
		args = append(args, "--no-models-autoload")
	}
	if cfg.Server.LookupCacheStatic != "" {
		add("--lookup-cache-static", cfg.Server.LookupCacheStatic)
	}
	if cfg.Server.LookupCacheDynamic != "" {
		add("--lookup-cache-dynamic", cfg.Server.LookupCacheDynamic)
	}

	// Endpoints
	addIf("--metrics", cfg.Server.Endpoints.Metrics)
	addIf("--no-slots", !*cfg.Server.Endpoints.Slots)
	addIf("--embedding", cfg.Server.Endpoints.Embeddings)
	addIf("--rerank", cfg.Server.Endpoints.Rerank)
	addIf("--props", cfg.Server.Endpoints.Props)

	// Shared args (hardware, context, sampling, chat, rope, model extras)
	args = appendSharedArgs(args, rc, false)

	// Server-only: show-timings
	if cfg.Logging.ShowTimings != nil && !*cfg.Logging.ShowTimings {
		args = append(args, "--no-show-timings")
	}

	return args
}

// appendSharedArgs appends flags common to both llama-server and llama-cli.
// interactive=true skips server-only flags and applies CLI-specific behaviour.
func appendSharedArgs(args []string, rc *config.RunConfig, interactive bool) []string {
	cfg := rc.Config
	p := rc.Profile

	add := func(flag string, value ...string) []string {
		args = append(args, flag)
		args = append(args, value...)
		return args
	}
	addIf := func(flag string, cond bool) {
		if cond {
			args = append(args, flag)
		}
	}

	// Hardware / GPU
	args = add("-ngl", strconv.Itoa(p.NGPULayers))
	if len(p.Devices) > 0 {
		args = add("--device", strings.Join(p.Devices, ","))
	}
	if len(p.TensorSplit) > 0 {
		parts := make([]string, len(p.TensorSplit))
		for i, v := range p.TensorSplit {
			parts[i] = fmt.Sprintf("%.2f", v)
		}
		args = add("--tensor-split", strings.Join(parts, ","))
	}
	if p.SplitMode != "" {
		args = add("--split-mode", p.SplitMode)
	}
	if p.MainGPU >= 0 {
		args = add("--main-gpu", strconv.Itoa(p.MainGPU))
	}
	if p.Priority != 0 {
		args = add("--prio", strconv.Itoa(p.Priority))
	}
	if p.Fit != "" {
		args = add("--fit", p.Fit)
	}
	if len(p.FitTarget) > 0 {
		parts := make([]string, len(p.FitTarget))
		for i, v := range p.FitTarget {
			parts[i] = strconv.Itoa(v)
		}
		args = add("--fit-target", strings.Join(parts, ","))
	}
	if p.FitCtx > 0 {
		args = add("--fit-ctx", strconv.Itoa(p.FitCtx))
	}
	for _, ot := range p.OverrideTensor {
		args = add("--override-tensor", ot)
	}
	addIf("--cpu-moe", p.CPUMoE)
	if p.CPUMask != "" {
		args = add("--cpu-mask", p.CPUMask)
	}
	if p.CPUMaskBatch != "" {
		args = add("--cpu-mask-batch", p.CPUMaskBatch)
	}
	if p.Poll != nil {
		args = add("--poll", strconv.Itoa(*p.Poll))
	}
	if p.PollBatch != nil {
		args = add("--poll-batch", strconv.Itoa(*p.PollBatch))
	}
	if p.PrioBatch != 0 {
		args = add("--prio-batch", strconv.Itoa(p.PrioBatch))
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
		args = add("--rpc", p.RPC)
	}
	if p.DirectIO != nil && *p.DirectIO {
		args = append(args, "--direct-io")
	}
	if p.CPURangeBatch != "" {
		args = add("--cpu-range-batch", p.CPURangeBatch)
	}
	if p.CPUStrictBatch {
		args = append(args, "--cpu-strict-batch", "1")
	}

	// Threads — interactive skips --threads on GPU backends
	if p.Threads > 0 {
		if !interactive || (!p.CUDA && !p.Metal && !p.ROCm) {
			args = add("--threads", strconv.Itoa(p.Threads))
		}
	}
	if p.ThreadsBatch > 0 {
		args = add("--threads-batch", strconv.Itoa(p.ThreadsBatch))
	}

	// Context
	ctx := cfg.Context
	args = add("--ctx-size", strconv.Itoa(ctx.NCtx))
	args = add("--batch-size", strconv.Itoa(ctx.NBatch))
	if ctx.NUBatch > 0 && ctx.NUBatch != ctx.NBatch {
		args = add("--ubatch-size", strconv.Itoa(ctx.NUBatch))
	}
	if ctx.NKeep > 0 {
		args = add("--keep", strconv.Itoa(ctx.NKeep))
	}
	if ctx.CacheTypeK != "" && ctx.CacheTypeK != "f16" {
		args = add("--cache-type-k", ctx.CacheTypeK)
	}
	if ctx.CacheTypeV != "" && ctx.CacheTypeV != "f16" {
		args = add("--cache-type-v", ctx.CacheTypeV)
	}
	addIf("--no-mmap", !*ctx.MMap)
	addIf("--mlock", ctx.MLock)
	if ctx.FlashAttention != "" {
		args = add("--flash-attn", ctx.FlashAttention)
	}
	if ctx.NCPUMoE > 0 {
		args = add("--n-cpu-moe", strconv.Itoa(ctx.NCPUMoE))
	}
	if ctx.NPredict != 0 {
		args = add("--predict", strconv.Itoa(ctx.NPredict))
	}
	addIf("--context-shift", ctx.ContextShift)
	if ctx.KVOffload != nil && !*ctx.KVOffload {
		args = append(args, "--no-kv-offload")
	}
	addIf("--swa-full", ctx.SWAFull)
	if ctx.CacheRAM != 0 {
		args = add("--cache-ram", strconv.Itoa(ctx.CacheRAM))
	}
	if ctx.ImageMinTokens > 0 {
		args = add("--image-min-tokens", strconv.Itoa(ctx.ImageMinTokens))
	}
	if ctx.ImageMaxTokens > 0 {
		args = add("--image-max-tokens", strconv.Itoa(ctx.ImageMaxTokens))
	}
	addIf("--check-tensors", ctx.CheckTensors)
	if ctx.CtxCheckpoints > 0 {
		args = add("--ctx-checkpoints", strconv.Itoa(ctx.CtxCheckpoints))
	}
	if ctx.CheckpointEveryNTokens != 0 {
		args = add("--checkpoint-every-n-tokens", strconv.Itoa(ctx.CheckpointEveryNTokens))
	}

	// Sampling
	s := cfg.Sampling
	if s.Temperature != nil {
		args = add("--temp", fmt.Sprintf("%.4f", *s.Temperature))
	}
	if s.TopK != nil {
		args = add("--top-k", strconv.Itoa(*s.TopK))
	}
	if s.TopP != nil {
		args = add("--top-p", fmt.Sprintf("%.4f", *s.TopP))
	}
	if s.MinP != nil {
		args = add("--min-p", fmt.Sprintf("%.4f", *s.MinP))
	}
	if s.RepeatPenalty != nil {
		args = add("--repeat-penalty", fmt.Sprintf("%.4f", *s.RepeatPenalty))
	}
	if s.RepeatLastN != nil {
		args = add("--repeat-last-n", strconv.Itoa(*s.RepeatLastN))
	}
	if s.PresencePenalty != 0 {
		args = add("--presence-penalty", fmt.Sprintf("%.4f", s.PresencePenalty))
	}
	if s.FrequencyPenalty != 0 {
		args = add("--frequency-penalty", fmt.Sprintf("%.4f", s.FrequencyPenalty))
	}
	if s.DryMultiplier > 0 {
		args = add("--dry-multiplier", fmt.Sprintf("%.4f", s.DryMultiplier))
		args = add("--dry-base", fmt.Sprintf("%.4f", s.DryBase))
		args = add("--dry-allowed-length", strconv.Itoa(s.DryAllowedLength))
		args = add("--dry-penalty-last-n", strconv.Itoa(s.DryPenaltyLastN))
	}
	if s.DynatempRange != 0 {
		args = add("--dynatemp-range", fmt.Sprintf("%.4f", s.DynatempRange))
		args = add("--dynatemp-exp", fmt.Sprintf("%.4f", s.DynatempExp))
	}
	if s.XTCProbability != 0 {
		args = add("--xtc-probability", fmt.Sprintf("%.4f", s.XTCProbability))
		args = add("--xtc-threshold", fmt.Sprintf("%.4f", s.XTCThreshold))
	}
	if s.Typical != 0 {
		args = add("--typical", fmt.Sprintf("%.4f", s.Typical))
	}
	if s.TopNSigma != 0 {
		args = add("--top-nsigma", fmt.Sprintf("%.4f", s.TopNSigma))
	}
	if s.AdaptiveTarget != 0 {
		args = add("--adaptive-target", fmt.Sprintf("%.4f", s.AdaptiveTarget))
		if s.AdaptiveDecay != 0 {
			args = add("--adaptive-decay", fmt.Sprintf("%.4f", s.AdaptiveDecay))
		}
	}
	for _, breaker := range s.DrySequenceBreakers {
		args = add("--dry-sequence-breaker", breaker)
	}
	if s.Mirostat > 0 {
		args = add("--mirostat", strconv.Itoa(s.Mirostat))
		if s.MirostatTau != nil {
			args = add("--mirostat-ent", fmt.Sprintf("%.4f", *s.MirostatTau))
		}
		if s.MirostatEta != nil {
			args = add("--mirostat-lr", fmt.Sprintf("%.4f", *s.MirostatEta))
		}
	}
	if s.Samplers != "" {
		args = add("--samplers", s.Samplers)
	}
	if s.SamplerSeq != "" {
		args = add("--sampler-seq", s.SamplerSeq)
	}
	addIf("--ignore-eos", s.IgnoreEOS)
	if s.Seed != 0 {
		args = add("--seed", strconv.FormatInt(s.Seed, 10))
	}
	if s.Grammar != "" {
		args = add("--grammar", s.Grammar)
	}
	if s.GrammarFile != "" {
		args = add("--grammar-file", s.GrammarFile)
	}
	if s.JSONSchema != "" {
		args = add("--json-schema", s.JSONSchema)
	}
	if s.JSONSchemaFile != "" {
		args = add("--json-schema-file", s.JSONSchemaFile)
	}
	addIf("--backend-sampling", s.BackendSampling)

	// Chat template
	// In interactive mode, skip --chat-template: the model's embedded template takes
	// precedence and explicit overrides break formatting in conversation mode.
	if !interactive && cfg.Chat.Template != "" {
		args = add("--chat-template", cfg.Chat.Template)
	}
	if cfg.Chat.SystemPrompt != "" {
		args = add("--system-prompt", cfg.Chat.SystemPrompt)
	}
	if cfg.Chat.Jinja != nil {
		if *cfg.Chat.Jinja {
			args = append(args, "--jinja")
		} else {
			args = append(args, "--no-jinja")
		}
	}
	if len(cfg.Chat.TemplateKwargs) > 0 {
		if b, err := json.Marshal(cfg.Chat.TemplateKwargs); err == nil {
			args = add("--chat-template-kwargs", string(b))
		}
	}
	if cfg.Chat.Reasoning != "" {
		args = add("--reasoning", cfg.Chat.Reasoning)
	}
	if cfg.Chat.ReasoningBudget != nil {
		args = add("--reasoning-budget", strconv.Itoa(*cfg.Chat.ReasoningBudget))
	}
	if cfg.Chat.ReasoningBudgetMessage != "" {
		args = add("--reasoning-budget-message", cfg.Chat.ReasoningBudgetMessage)
	}
	if cfg.Chat.ReasoningFormat != "" {
		args = add("--reasoning-format", cfg.Chat.ReasoningFormat)
	}
	if cfg.Chat.TemplateFile != "" {
		args = add("--chat-template-file", cfg.Chat.TemplateFile)
	}
	addIf("--skip-chat-parsing", cfg.Chat.SkipChatParsing)

	// RoPE
	rope := cfg.Rope
	if rope.Scaling != "" {
		args = add("--rope-scaling", rope.Scaling)
	}
	if rope.Scale > 0 {
		args = add("--rope-scale", fmt.Sprintf("%.6f", rope.Scale))
	}
	if rope.FreqBase > 0 {
		args = add("--rope-freq-base", fmt.Sprintf("%.1f", rope.FreqBase))
	}
	if rope.FreqScale > 0 && rope.FreqScale != 1.0 {
		args = add("--rope-freq-scale", fmt.Sprintf("%.6f", rope.FreqScale))
	}
	if rope.Scaling == "yarn" {
		args = add("--yarn-ext-factor", fmt.Sprintf("%.4f", rope.YarnExtFactor))
		args = add("--yarn-attn-factor", fmt.Sprintf("%.4f", rope.YarnAttnFactor))
		if rope.YarnBetaSlow != 0 {
			args = add("--yarn-beta-slow", fmt.Sprintf("%.4f", rope.YarnBetaSlow))
		}
		if rope.YarnBetaFast != 0 {
			args = add("--yarn-beta-fast", fmt.Sprintf("%.4f", rope.YarnBetaFast))
		}
		if rope.YarnOrigCtx > 0 {
			args = add("--yarn-orig-ctx", strconv.Itoa(rope.YarnOrigCtx))
		}
	}

	// Draft model (speculative decoding)
	if rc.DraftModelPath != "" {
		args = add("--model-draft", rc.DraftModelPath)
		if d := cfg.Model.Draft; d != nil {
			if d.DraftN > 0 {
				args = add("--draft", strconv.Itoa(d.DraftN))
			}
			if d.DraftMin > 0 {
				args = add("--draft-min", strconv.Itoa(d.DraftMin))
			}
			if d.DraftPMin > 0 {
				args = add("--draft-p-min", fmt.Sprintf("%.4f", d.DraftPMin))
			}
			if d.NCtx > 0 {
				args = add("--ctx-size-draft", strconv.Itoa(d.NCtx))
			}
			if d.NGPULayers > 0 {
				args = add("--n-gpu-layers-draft", strconv.Itoa(d.NGPULayers))
			}
			if len(d.Devices) > 0 {
				args = add("--device-draft", strings.Join(d.Devices, ","))
			}
			if d.CacheTypeK != "" && d.CacheTypeK != "f16" {
				args = add("--cache-type-k-draft", d.CacheTypeK)
			}
			if d.CacheTypeV != "" && d.CacheTypeV != "f16" {
				args = add("--cache-type-v-draft", d.CacheTypeV)
			}
			if d.SpecReplaceTarget != "" && d.SpecReplaceDraft != "" {
				args = add("--spec-replace", d.SpecReplaceTarget, d.SpecReplaceDraft)
			}
			for _, ot := range d.OverrideTensor {
				args = add("--override-tensor-draft", ot)
			}
			if d.CPUMoE {
				args = append(args, "--cpu-moe-draft")
			}
			if d.NCPUMoE > 0 {
				args = add("--n-cpu-moe-draft", strconv.Itoa(d.NCPUMoE))
			}
			if d.ThreadsDraft > 0 {
				args = add("--threads-draft", strconv.Itoa(d.ThreadsDraft))
			}
			if d.ThreadsBatchDraft > 0 {
				args = add("--threads-batch-draft", strconv.Itoa(d.ThreadsBatchDraft))
			}
			if d.SpecType != "" {
				args = add("--spec-type", d.SpecType)
			}
			if d.SpecNgramSizeN > 0 {
				args = add("--spec-ngram-size-n", strconv.Itoa(d.SpecNgramSizeN))
			}
			if d.SpecNgramSizeM > 0 {
				args = add("--spec-ngram-size-m", strconv.Itoa(d.SpecNgramSizeM))
			}
			if d.SpecNgramMinHits > 0 {
				args = add("--spec-ngram-min-hits", strconv.Itoa(d.SpecNgramMinHits))
			}
		}
	}

	// LoRA
	for _, lora := range cfg.Model.LoRA {
		args = add("--lora", lora)
	}
	if len(cfg.Model.LoRAScaled) > 0 {
		args = add("--lora-scaled", strings.Join(cfg.Model.LoRAScaled, ","))
	}

	// Control vectors
	for _, cv := range cfg.Model.ControlVector {
		args = add("--control-vector", cv)
	}
	if len(cfg.Model.ControlVectorScaled) > 0 {
		args = add("--control-vector-scaled", strings.Join(cfg.Model.ControlVectorScaled, ","))
	}
	if cfg.Model.ControlVectorLayerStart >= 0 && cfg.Model.ControlVectorLayerEnd >= 0 {
		args = add("--control-vector-layer-range",
			strconv.Itoa(cfg.Model.ControlVectorLayerStart),
			strconv.Itoa(cfg.Model.ControlVectorLayerEnd))
	}

	// Model metadata overrides
	for _, kv := range cfg.Model.OverrideKV {
		args = add("--override-kv", kv)
	}

	// Multimodal projection
	if rc.MMProjPath != "" {
		args = add("--mmproj", rc.MMProjPath)
		if cfg.Model.MMProj != nil && cfg.Model.MMProj.Offload != nil && !*cfg.Model.MMProj.Offload {
			args = append(args, "--no-mmproj-offload")
		}
	}

	// CPU-specific
	if p.CPURange != "" {
		args = add("--cpu-range", p.CPURange)
	}
	if p.CPUStrict {
		args = append(args, "--cpu-strict", "1")
	}
	if p.NUMA != "" {
		args = add("--numa", p.NUMA)
	}

	// Logging
	log := cfg.Logging
	if log.File != "" {
		args = add("--log-file", log.File)
	}
	if log.Colors != "" && log.Colors != "auto" {
		args = add("--log-colors", log.Colors)
	}
	addIf("--log-prefix", log.Prefix)
	addIf("--log-timestamps", log.Timestamps)
	if log.Verbosity >= 0 {
		args = add("--log-verbosity", strconv.Itoa(log.Verbosity))
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
