package config

import (
	"fmt"
	"strings"
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
	case "":
		errs = append(errs, "model.source is required (huggingface | local | url)")
	default:
		errs = append(errs, fmt.Sprintf("model.source %q is invalid (huggingface | local | url)", cfg.Model.Source))
	}

	if cfg.Mode != "" && cfg.Mode != "server" && cfg.Mode != "interactive" {
		errs = append(errs, fmt.Sprintf("mode %q is invalid (server | interactive)", cfg.Mode))
	}

	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port %d is out of range", cfg.Server.Port))
	}

	if cfg.Server.Parallel < 0 {
		errs = append(errs, "server.parallel must be >= 0")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
