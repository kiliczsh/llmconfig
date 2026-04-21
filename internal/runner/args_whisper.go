package runner

import (
	"fmt"
	"strconv"

	"github.com/kiliczsh/llamaconfig/internal/config"
)

func buildWhisperArgs(rc *config.RunConfig) []string {
	cfg := rc.Config
	p := rc.Profile
	var args []string

	args = append(args, "--model", rc.ModelPath)

	if cfg.Mode == "server" || cfg.Mode == "" {
		args = append(args, "--host", cfg.Server.Host)
		args = append(args, "--port", strconv.Itoa(cfg.Server.Port))
	}

	if p.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(p.Threads))
	}
	if cfg.Whisper.Processors > 0 {
		args = append(args, "--processors", strconv.Itoa(cfg.Whisper.Processors))
	}
	if cfg.Whisper.Language != "" && cfg.Whisper.Language != "auto" {
		args = append(args, "--language", cfg.Whisper.Language)
	}
	if cfg.Whisper.Task == "translate" {
		args = append(args, "--translate")
	}
	if cfg.Whisper.BeamSize > 0 {
		args = append(args, "--beam-size", strconv.Itoa(cfg.Whisper.BeamSize))
	}
	if cfg.Whisper.BestOf > 0 {
		args = append(args, "--best-of", strconv.Itoa(cfg.Whisper.BestOf))
	}
	if cfg.Whisper.VAD {
		args = append(args, "--vad")
		if cfg.Whisper.VADThreshold > 0 {
			args = append(args, "--vad-threshold", fmt.Sprintf("%.2f", cfg.Whisper.VADThreshold))
		}
	}
	if cfg.Whisper.WordTimestamps {
		// word-level timestamps via dynamic time warping
		args = append(args, "--dtw", "tiny")
	}

	return args
}
