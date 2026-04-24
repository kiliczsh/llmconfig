package runner

import (
	"fmt"
	"strconv"

	"github.com/kiliczsh/llamaconfig/internal/config"
)

func buildSDArgs(rc *config.RunConfig) []string {
	cfg := rc.Config
	sd := cfg.SD
	p := rc.Profile
	var args []string

	add := func(flag string, val string) { args = append(args, flag, val) }
	addIf := func(flag string, cond bool) {
		if cond {
			args = append(args, flag)
		}
	}

	// Main weights. sd.cpp takes a unified checkpoint via --model for
	// SD1.x/SDXL, but Flux and SD3 need the diffusion weights routed
	// through --diffusion-model so the encoders (clip_l/t5xxl/vae) can
	// be loaded separately. Signal: user set an explicit diffusion_model,
	// or supplied a separate text encoder (clip_l or t5xxl).
	switch {
	case sd.DiffusionModel != "":
		// Explicit diffusion_model wins; skip --model to avoid sd.cpp
		// trying to detect an SD version from the main gguf.
	case sd.ClipL != "" || sd.T5XXL != "":
		add("--diffusion-model", rc.ModelPath)
	default:
		add("--model", rc.ModelPath)
	}

	// Server vs CLI mode
	if cfg.Mode == "server" {
		add("--listen-ip", cfg.Server.Host)
		add("--listen-port", strconv.Itoa(cfg.Server.Port))
		if sd.ServeHTMLPath != "" {
			add("--serve-html-path", sd.ServeHTMLPath)
		}
	} else {
		// CLI output & preview
		if sd.Output != "" {
			add("--output", sd.Output)
		}
		if sd.PreviewPath != "" {
			add("--preview-path", sd.PreviewPath)
		}
		if sd.PreviewInterval > 0 {
			add("--preview-interval", strconv.Itoa(sd.PreviewInterval))
		}
		if sd.Preview != "" {
			add("--preview", sd.Preview)
		}
		if sd.Mode != "" {
			add("--mode", sd.Mode)
		}
	}

	// Threads
	if p.Threads > 0 {
		add("-t", strconv.Itoa(p.Threads))
	}

	// Model components
	if sd.ClipL != "" {
		add("--clip_l", sd.ClipL)
	}
	if sd.ClipG != "" {
		add("--clip_g", sd.ClipG)
	}
	if sd.ClipVision != "" {
		add("--clip_vision", sd.ClipVision)
	}
	if sd.T5XXL != "" {
		add("--t5xxl", sd.T5XXL)
	}
	if sd.LLM != "" {
		add("--llm", sd.LLM)
	}
	if sd.LLMVision != "" {
		add("--llm_vision", sd.LLMVision)
	}
	if sd.DiffusionModel != "" {
		add("--diffusion-model", sd.DiffusionModel)
	}
	if sd.HighNoiseDiffusionModel != "" {
		add("--high-noise-diffusion-model", sd.HighNoiseDiffusionModel)
	}
	if sd.VAE != "" {
		add("--vae", sd.VAE)
	}
	if sd.TAESD != "" {
		add("--taesd", sd.TAESD)
	}
	if sd.ControlNet != "" {
		add("--control-net", sd.ControlNet)
	}
	if sd.EmbedDir != "" {
		add("--embd-dir", sd.EmbedDir)
	}
	if sd.LoRAModelDir != "" {
		add("--lora-model-dir", sd.LoRAModelDir)
	}
	if sd.TensorTypeRules != "" {
		add("--tensor-type-rules", sd.TensorTypeRules)
	}
	if sd.PhotoMaker != "" {
		add("--photo-maker", sd.PhotoMaker)
	}
	if sd.UpscaleModel != "" {
		add("--upscale-model", sd.UpscaleModel)
	}

	// Hardware / context options
	if sd.Type != "" {
		add("--type", sd.Type)
	}
	if sd.RNG != "" {
		add("--rng", sd.RNG)
	}
	if sd.SamplerRNG != "" {
		add("--sampler-rng", sd.SamplerRNG)
	}
	if sd.Prediction != "" {
		add("--prediction", sd.Prediction)
	}
	if sd.LoRAApplyMode != "" {
		add("--lora-apply-mode", sd.LoRAApplyMode)
	}
	addIf("--offload-to-cpu", sd.OffloadToCPU)
	addIf("--mmap", sd.MMap)
	addIf("--control-net-cpu", sd.ControlNetCPU)
	addIf("--clip-on-cpu", sd.ClipOnCPU)
	addIf("--vae-on-cpu", sd.VAEOnCPU)
	addIf("--fa", sd.FlashAttention)
	addIf("--diffusion-fa", sd.DiffusionFA)
	addIf("--diffusion-conv-direct", sd.DiffusionConvDirect)
	addIf("--vae-conv-direct", sd.VAEConvDirect)
	addIf("--circular", sd.Circular)
	addIf("--circularx", sd.CircularX)
	addIf("--circulary", sd.CircularY)

	// Generation
	if sd.Width > 0 {
		add("-W", strconv.Itoa(sd.Width))
	}
	if sd.Height > 0 {
		add("-H", strconv.Itoa(sd.Height))
	}
	if sd.Steps > 0 {
		add("--steps", strconv.Itoa(sd.Steps))
	}
	if sd.HighNoiseSteps != 0 {
		add("--high-noise-steps", strconv.Itoa(sd.HighNoiseSteps))
	}
	if sd.ClipSkip != 0 {
		add("--clip-skip", strconv.Itoa(sd.ClipSkip))
	}
	if sd.BatchCount > 0 {
		add("-b", strconv.Itoa(sd.BatchCount))
	}
	if sd.VideoFrames > 0 {
		add("--video-frames", strconv.Itoa(sd.VideoFrames))
	}
	if sd.FPS > 0 {
		add("--fps", strconv.Itoa(sd.FPS))
	}
	if sd.TimestepShift != 0 {
		add("--timestep-shift", strconv.Itoa(sd.TimestepShift))
	}
	if sd.UpscaleRepeats > 0 {
		add("--upscale-repeats", strconv.Itoa(sd.UpscaleRepeats))
	}
	if sd.UpscaleTileSize > 0 {
		add("--upscale-tile-size", strconv.Itoa(sd.UpscaleTileSize))
	}
	if sd.CFGScale > 0 {
		add("--cfg-scale", fmt.Sprintf("%.2f", sd.CFGScale))
	}
	if sd.ImgCFGScale > 0 {
		add("--img-cfg-scale", fmt.Sprintf("%.2f", sd.ImgCFGScale))
	}
	if sd.Guidance > 0 {
		add("--guidance", fmt.Sprintf("%.2f", sd.Guidance))
	}
	if sd.SLGScale > 0 {
		add("--slg-scale", fmt.Sprintf("%.4f", sd.SLGScale))
		if sd.SkipLayerStart != 0 {
			add("--skip-layer-start", fmt.Sprintf("%.4f", sd.SkipLayerStart))
		}
		if sd.SkipLayerEnd != 0 {
			add("--skip-layer-end", fmt.Sprintf("%.4f", sd.SkipLayerEnd))
		}
		if sd.SkipLayers != "" {
			add("--skip-layers", sd.SkipLayers)
		}
	}
	if sd.Eta != 0 {
		add("--eta", fmt.Sprintf("%.4f", sd.Eta))
	}
	if sd.FlowShift != 0 {
		add("--flow-shift", fmt.Sprintf("%.4f", sd.FlowShift))
	}
	if sd.Strength > 0 {
		add("--strength", fmt.Sprintf("%.4f", sd.Strength))
	}
	if sd.ControlStrength > 0 {
		add("--control-strength", fmt.Sprintf("%.4f", sd.ControlStrength))
	}
	if sd.VAETileOverlap > 0 {
		add("--vae-tile-overlap", fmt.Sprintf("%.4f", sd.VAETileOverlap))
	}
	if sd.Seed != 0 {
		add("-s", strconv.FormatInt(sd.Seed, 10))
	}
	if sd.SamplingMethod != "" {
		add("--sampling-method", sd.SamplingMethod)
	}
	if sd.HighNoiseSamplingMethod != "" {
		add("--high-noise-sampling-method", sd.HighNoiseSamplingMethod)
	}
	if sd.Scheduler != "" {
		add("--scheduler", sd.Scheduler)
	}
	if sd.NegativePrompt != "" {
		add("-n", sd.NegativePrompt)
	}
	if sd.Sigmas != "" {
		add("--sigmas", sd.Sigmas)
	}
	if sd.RefImage != "" {
		add("-r", sd.RefImage)
	}
	addIf("--vae-tiling", sd.VAETiling)
	if sd.VAETileSize != "" {
		add("--vae-tile-size", sd.VAETileSize)
	}
	if sd.VAERelativeTileSize != "" {
		add("--vae-relative-tile-size", sd.VAERelativeTileSize)
	}
	addIf("--disable-image-metadata", sd.DisableImageMetadata)

	// High noise stage
	if sd.HighNoiseCFGScale > 0 {
		add("--high-noise-cfg-scale", fmt.Sprintf("%.2f", sd.HighNoiseCFGScale))
	}
	if sd.HighNoiseImgCFGScale > 0 {
		add("--high-noise-img-cfg-scale", fmt.Sprintf("%.2f", sd.HighNoiseImgCFGScale))
	}
	if sd.HighNoiseGuidance > 0 {
		add("--high-noise-guidance", fmt.Sprintf("%.2f", sd.HighNoiseGuidance))
	}
	if sd.HighNoiseSLGScale > 0 {
		add("--high-noise-slg-scale", fmt.Sprintf("%.4f", sd.HighNoiseSLGScale))
		if sd.HighNoiseSkipLayerStart != 0 {
			add("--high-noise-skip-layer-start", fmt.Sprintf("%.4f", sd.HighNoiseSkipLayerStart))
		}
		if sd.HighNoiseSkipLayerEnd != 0 {
			add("--high-noise-skip-layer-end", fmt.Sprintf("%.4f", sd.HighNoiseSkipLayerEnd))
		}
		if sd.HighNoiseSkipLayers != "" {
			add("--high-noise-skip-layers", sd.HighNoiseSkipLayers)
		}
	}
	if sd.HighNoiseEta != 0 {
		add("--high-noise-eta", fmt.Sprintf("%.4f", sd.HighNoiseEta))
	}

	// Cache
	if sd.CacheMode != "" {
		add("--cache-mode", sd.CacheMode)
	}
	if sd.CacheOption != "" {
		add("--cache-option", sd.CacheOption)
	}
	if sd.SCMMask != "" {
		add("--scm-mask", sd.SCMMask)
	}
	if sd.SCMPolicy != "" {
		add("--scm-policy", sd.SCMPolicy)
	}

	// Logging
	addIf("-v", sd.Verbose)
	addIf("--color", sd.Color)

	return args
}
