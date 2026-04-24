package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/dirs"
	"github.com/kiliczsh/llmconfig/internal/hardware"
)

// Resolve merges the detected hardware profile into a RunConfig.
// modelPath must already be resolved (downloaded or local).
func Resolve(cfg *Config, hw *hardware.DetectionResult, binaryPath string) (*RunConfig, error) {
	profile, profileName := selectProfile(cfg, hw)

	modelPath, err := resolveModelPath(cfg)
	if err != nil {
		return nil, err
	}

	logFile := cfg.Logging.File
	if logFile == "" {
		logFile = filepath.Join(dirs.LogDir(), cfg.Name+".log")
	} else {
		logFile = dirs.ExpandHome(logFile)
	}

	mmprojPath, err := resolveMMProjPath(cfg)
	if err != nil {
		return nil, err
	}

	rc := &RunConfig{
		Config:      cfg,
		ModelPath:   modelPath,
		MMProjPath:  mmprojPath,
		Profile:     profile,
		ProfileName: profileName,
		LogFile:     logFile,
		BinaryPath:  binaryPath,
		Backend:     cfg.Backend,
	}

	if cfg.Model.Draft != nil {
		rc.DraftModelPath, err = resolveDraftPath(cfg)
		if err != nil {
			return nil, err
		}
	}

	rc.ExtraDownloads = resolveSDAux(cfg)

	return rc, nil
}

// resolveSDAux expands `hf://<owner>/<repo>/<file>` references in every
// sd.* auxiliary path field: the string is rewritten in-place to the
// local cache path, and an ExtraDownload entry is emitted so `up` can
// fetch the file before starting sd-server. Tilde prefixes on plain
// filesystem paths are also expanded here so args_sd.go can forward them
// verbatim.
func resolveSDAux(cfg *Config) []ExtraDownload {
	if cfg.Backend != "sd" {
		return nil
	}
	cache := modelDir(cfg)
	var extras []ExtraDownload

	resolveOne := func(kind string, val *string) {
		if *val == "" {
			return
		}
		if repo, file, ok := parseHFRef(*val); ok {
			dest := filepath.Join(cache, file)
			extras = append(extras, ExtraDownload{
				Kind:     kind,
				Repo:     repo,
				File:     file,
				DestPath: dest,
			})
			*val = dest
			return
		}
		*val = dirs.ExpandHome(*val)
	}

	resolveOne("vae", &cfg.SD.VAE)
	resolveOne("taesd", &cfg.SD.TAESD)
	resolveOne("clip_l", &cfg.SD.ClipL)
	resolveOne("clip_g", &cfg.SD.ClipG)
	resolveOne("clip_vision", &cfg.SD.ClipVision)
	resolveOne("t5xxl", &cfg.SD.T5XXL)
	resolveOne("llm", &cfg.SD.LLM)
	resolveOne("llm_vision", &cfg.SD.LLMVision)
	resolveOne("diffusion_model", &cfg.SD.DiffusionModel)
	resolveOne("high_noise_diffusion_model", &cfg.SD.HighNoiseDiffusionModel)
	resolveOne("control_net", &cfg.SD.ControlNet)
	resolveOne("photo_maker", &cfg.SD.PhotoMaker)
	resolveOne("upscale_model", &cfg.SD.UpscaleModel)
	resolveOne("embd_dir", &cfg.SD.EmbedDir)
	resolveOne("lora_model_dir", &cfg.SD.LoRAModelDir)

	return extras
}

// parseHFRef parses an "hf://owner/name/path/to/file" string. Anything
// missing the full three-segment shape is returned as not-an-hf-ref so
// the caller falls back to treating the value as a filesystem path.
func parseHFRef(s string) (repo, file string, ok bool) {
	rest, found := strings.CutPrefix(s, "hf://")
	if !found {
		return "", "", false
	}
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[0] + "/" + parts[1], parts[2], true
}

// resolveModelPath returns the path where the main model file should live.
// It does not check existence: validator is responsible for that when
// source=local, and the downloader fetches it when source=huggingface|url.
// Keeping the resolver lenient lets `up --dry-run` and `inspect` work
// before the file is on disk.
func resolveModelPath(cfg *Config) (string, error) {
	switch cfg.Model.Source {
	case "local":
		return dirs.ExpandHome(cfg.Model.Path), nil
	case "huggingface", "url":
		return filepath.Join(modelDir(cfg), cfg.Model.File), nil
	default:
		return "", fmt.Errorf("unknown model source: %s", cfg.Model.Source)
	}
}

func resolveDraftPath(cfg *Config) (string, error) {
	if cfg.Model.Draft == nil || cfg.Model.Draft.File == "" {
		return "", nil
	}
	if cfg.Model.Draft.Source == "local" {
		return dirs.ExpandHome(cfg.Model.Draft.File), nil
	}
	return filepath.Join(modelDir(cfg), cfg.Model.Draft.File), nil
}

func resolveMMProjPath(cfg *Config) (string, error) {
	if cfg.Model.MMProj == nil || cfg.Model.MMProj.File == "" {
		return "", nil
	}
	if cfg.Model.MMProj.Source == "local" {
		return dirs.ExpandHome(cfg.Model.MMProj.File), nil
	}
	return filepath.Join(modelDir(cfg), cfg.Model.MMProj.File), nil
}

func modelDir(cfg *Config) string {
	modelDir := cfg.Model.Download.ModelDir
	if modelDir == "" {
		return dirs.ModelsDir()
	}
	return dirs.ExpandHome(modelDir)
}

// selectProfile picks the best HardwareProfile from the config.
// Lives here (not in hardware package) to avoid import cycle.
func selectProfile(cfg *Config, hw *hardware.DetectionResult) (HardwareProfile, string) {
	profiles := cfg.HardwareProfiles

	switch hw.Class {
	case hardware.ClassAppleSilicon:
		p := profiles.AppleSilicon
		if isEmptyProfile(p) {
			p = HardwareProfile{NGPULayers: 99, Metal: true, Threads: -1}
		}
		p.Metal = true
		return p, "apple_silicon"

	case hardware.ClassNVIDIA:
		p := profiles.NVIDIA
		if isEmptyProfile(p) {
			p = HardwareProfile{NGPULayers: 99, CUDA: true, Threads: -1}
		}
		p.CUDA = true
		return p, "nvidia"

	case hardware.ClassAMD:
		p := profiles.AMD
		if isEmptyProfile(p) {
			p = HardwareProfile{NGPULayers: 99, ROCm: true, Threads: -1}
		}
		p.ROCm = true
		return p, "amd"

	case hardware.ClassIntelGPU:
		p := profiles.IntelGPU
		if isEmptyProfile(p) {
			p = HardwareProfile{NGPULayers: 99, SYCL: true, Threads: -1}
		}
		return p, "intel_gpu"

	default:
		p := profiles.CPU
		if isEmptyProfile(p) {
			p = HardwareProfile{NGPULayers: 0, Threads: -1}
		}
		return p, "cpu"
	}
}

func isEmptyProfile(p HardwareProfile) bool {
	return p.NGPULayers == 0 && !p.Metal && !p.CUDA && !p.ROCm && !p.SYCL && p.Threads == 0
}
