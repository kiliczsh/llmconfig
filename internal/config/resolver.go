package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kiliczsh/llamaconfig/internal/dirs"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
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

	return rc, nil
}

func resolveModelPath(cfg *Config) (string, error) {
	switch cfg.Model.Source {
	case "local":
		p := dirs.ExpandHome(cfg.Model.Path)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("model file not found: %s", p)
		}
		return p, nil
	case "huggingface", "url":
		cacheDir := modelCacheDir(cfg)
		p := filepath.Join(cacheDir, cfg.Model.File)
		if _, err := os.Stat(p); err != nil {
			return p, nil // may not exist yet; downloader will fetch
		}
		return p, nil
	default:
		return "", fmt.Errorf("unknown model source: %s", cfg.Model.Source)
	}
}

func resolveDraftPath(cfg *Config) (string, error) {
	if cfg.Model.Draft == nil || cfg.Model.Draft.File == "" {
		return "", nil
	}

	switch cfg.Model.Draft.Source {
	case "local":
		p := dirs.ExpandHome(cfg.Model.Draft.File)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("draft model file not found: %s", p)
		}
		return p, nil
	default:
		return filepath.Join(modelCacheDir(cfg), cfg.Model.Draft.File), nil
	}
}

func resolveMMProjPath(cfg *Config) (string, error) {
	if cfg.Model.MMProj == nil || cfg.Model.MMProj.File == "" {
		return "", nil
	}

	switch cfg.Model.MMProj.Source {
	case "local":
		p := dirs.ExpandHome(cfg.Model.MMProj.File)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("mmproj file not found: %s", p)
		}
		return p, nil
	case "huggingface", "url", "":
		return filepath.Join(modelCacheDir(cfg), cfg.Model.MMProj.File), nil
	default:
		return filepath.Join(modelCacheDir(cfg), cfg.Model.MMProj.File), nil
	}
}

func modelCacheDir(cfg *Config) string {
	cacheDir := cfg.Model.Download.CacheDir
	if cacheDir == "" {
		return dirs.CacheDir()
	}
	return dirs.ExpandHome(cacheDir)
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
