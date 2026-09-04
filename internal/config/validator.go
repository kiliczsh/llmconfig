package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/kiliczsh/llmconfig/internal/dirs"
)

func Validate(cfg *Config) error {
	var errs []string

	if cfg.Version != 1 {
		errs = append(errs, fmt.Sprintf("version must be 1, got %d", cfg.Version))
	}
	if cfg.Name == "" {
		errs = append(errs, "name is required")
	}
	if strings.ContainsAny(cfg.Name, " \t\n/\\") {
		errs = append(errs, "name must not contain spaces or path separators")
	}

	switch cfg.Model.Source {
	case "huggingface":
		if cfg.Model.Repo == "" {
			errs = append(errs, "model.repo is required for source=huggingface")
		}
		if cfg.Model.File == "" {
			errs = append(errs, "model.file is required for source=huggingface")
		}
	case "local":
		if cfg.Model.Path == "" {
			errs = append(errs, "model.path is required for source=local")
		}
	case "url":
		if cfg.Model.URL == "" {
			errs = append(errs, "model.url is required for source=url")
		}
		if cfg.Model.File == "" {
			errs = append(errs, "model.file is required for source=url (filename to store in cache)")
		}
	case "":
		errs = append(errs, "model.source is required (huggingface | local | url)")
	default:
		errs = append(errs, fmt.Sprintf("model.source %q is invalid (huggingface | local | url)", cfg.Model.Source))
	}

	switch cfg.Backend {
	case "", "llama", "ik_llama", "sd", "whisper":
		// valid
	default:
		errs = append(errs, fmt.Sprintf("backend %q is invalid (llama | ik_llama | sd | whisper)", cfg.Backend))
	}

	validModes := map[string]bool{"": true, "server": true, "interactive": true}
	if cfg.Backend == "whisper" {
		validModes["stream"] = true
	}
	if !validModes[cfg.Mode] {
		if cfg.Backend == "whisper" {
			errs = append(errs, fmt.Sprintf("mode %q is invalid for whisper backend (server | interactive | stream)", cfg.Mode))
		} else {
			errs = append(errs, fmt.Sprintf("mode %q is invalid (server | interactive)", cfg.Mode))
		}
	}

	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port %d is out of range", cfg.Server.Port))
	}

	if cfg.Server.Parallel < 0 {
		errs = append(errs, "server.parallel must be >= 0")
	}

	validateExistingPath(&errs, "sampling.grammar_file", cfg.Sampling.GrammarFile)
	validateExistingPath(&errs, "sampling.json_schema_file", cfg.Sampling.JSONSchemaFile)
	validateExistingPath(&errs, "server.api_key_file", cfg.Server.APIKeyFile)
	validateExistingPath(&errs, "server.ssl_cert_file", cfg.Server.SSLCertFile)
	validateExistingPath(&errs, "server.ssl_key_file", cfg.Server.SSLKeyFile)
	validateExistingPath(&errs, "chat.template_file", cfg.Chat.TemplateFile)
	if cfg.Model.Source == "local" {
		validateExistingPath(&errs, "model.path", cfg.Model.Path)
	}
	if cfg.Model.Draft != nil && cfg.Model.Draft.Source == "local" {
		validateExistingPath(&errs, "model.draft.file", cfg.Model.Draft.File)
	}
	if cfg.Model.MMProj != nil && cfg.Model.MMProj.Source == "local" {
		validateExistingPath(&errs, "model.mmproj.file", cfg.Model.MMProj.File)
	}

	switch cfg.Backend {
	case "llama", "ik_llama", "":
		validateLlama(cfg, &errs)
	case "sd":
		validateSD(cfg, &errs)
	case "whisper":
		validateWhisper(cfg, &errs)
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func validateLlama(cfg *Config, errs *[]string) {
}

func validateSD(cfg *Config, errs *[]string) {
	validModes := map[string]bool{
		"":          true,
		"img_gen":   true,
		"adetailer": true,
		"vid_gen":   true,
		"upscale":   true,
		"convert":   true,
		"metadata":  true,
	}
	if !validModes[cfg.SD.Mode] {
		*errs = append(*errs, fmt.Sprintf("sd.mode %q is invalid (img_gen | adetailer | vid_gen | upscale | convert | metadata)", cfg.SD.Mode))
	}

	if cfg.SD.EndImage != "" && cfg.SD.InitImage == "" {
		*errs = append(*errs, "sd.end_image requires sd.init_image")
	}
	if len(cfg.SD.RefVideoAudios) > len(cfg.SD.RefVideos) {
		*errs = append(*errs, "sd.ref_video_audios cannot contain more entries than sd.ref_videos")
	}

	isMiniMaxH3 := false
	for _, value := range []string{cfg.Model.Repo, cfg.Model.File, cfg.Model.Path, cfg.Model.URL, cfg.SD.DiffusionModel} {
		value = strings.ToLower(value)
		if strings.Contains(value, "minimax-h3") || strings.Contains(value, "minimax_h3") {
			isMiniMaxH3 = true
			break
		}
	}
	if isMiniMaxH3 {
		if cfg.SD.Mode != "vid_gen" {
			*errs = append(*errs, "MiniMax-H3 requires sd.mode: vid_gen")
		}
		if cfg.SD.CFGScale > 1.0 {
			*errs = append(*errs, "MiniMax-H3 requires sd.cfg_scale <= 1.0")
		}
		if cfg.SD.LLM == "" {
			*errs = append(*errs, "MiniMax-H3 requires sd.llm")
		}
		if cfg.SD.VAE == "" {
			*errs = append(*errs, "MiniMax-H3 requires sd.vae")
		}
	}
}

func validateWhisper(cfg *Config, errs *[]string) {
	if cfg.Whisper.Task != "" && cfg.Whisper.Task != "transcribe" && cfg.Whisper.Task != "translate" {
		*errs = append(*errs, fmt.Sprintf("whisper.task %q is invalid (transcribe | translate)", cfg.Whisper.Task))
	}
	if cfg.Whisper.Language != "" && cfg.Whisper.Language != "auto" && len(cfg.Whisper.Language) < 2 {
		*errs = append(*errs, fmt.Sprintf("whisper.language %q is invalid (expected 2-char code or auto)", cfg.Whisper.Language))
	}
}

func validateExistingPath(errs *[]string, yamlPath, path string) {
	if path == "" {
		return
	}

	expanded := dirs.ExpandHome(path)
	if _, err := os.Stat(expanded); err != nil {
		if os.IsNotExist(err) {
			*errs = append(*errs, fmt.Sprintf("%s %q does not exist", yamlPath, path))
			return
		}
		*errs = append(*errs, fmt.Sprintf("%s %q: %v", yamlPath, path, err))
	}
}
