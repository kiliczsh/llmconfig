package runner

import (
	"fmt"
	"strconv"

	"github.com/kiliczsh/llamaconfig/internal/config"
)

func buildSDArgs(rc *config.RunConfig) []string {
	cfg := rc.Config
	p := rc.Profile
	var args []string

	args = append(args, "--model", rc.ModelPath)

	if cfg.Mode == "server" || cfg.Mode == "" {
		args = append(args, "--listen-ip", cfg.Server.Host)
		args = append(args, "--listen-port", strconv.Itoa(cfg.Server.Port))
	}

	if p.Threads > 0 {
		args = append(args, "-t", strconv.Itoa(p.Threads))
	}
	if cfg.SD.Width > 0 {
		args = append(args, "-W", strconv.Itoa(cfg.SD.Width))
	}
	if cfg.SD.Height > 0 {
		args = append(args, "-H", strconv.Itoa(cfg.SD.Height))
	}
	if cfg.SD.Steps > 0 {
		args = append(args, "--steps", strconv.Itoa(cfg.SD.Steps))
	}
	if cfg.SD.CFGScale > 0 {
		args = append(args, "--cfg-scale", fmt.Sprintf("%.1f", cfg.SD.CFGScale))
	}
	if cfg.SD.SamplingMethod != "" {
		args = append(args, "--sampling-method", cfg.SD.SamplingMethod)
	}
	if cfg.SD.Seed != 0 {
		args = append(args, "--seed", strconv.FormatInt(cfg.SD.Seed, 10))
	}

	return args
}
