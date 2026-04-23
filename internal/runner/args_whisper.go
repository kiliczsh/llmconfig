package runner

import (
	"fmt"
	"strconv"

	"github.com/kiliczsh/llamaconfig/internal/config"
)

func buildWhisperArgs(rc *config.RunConfig) []string {
	cfg := rc.Config
	w := cfg.Whisper
	p := rc.Profile
	var args []string

	add := func(flag string, val string) { args = append(args, flag, val) }
	addIf := func(flag string, cond bool) {
		if cond {
			args = append(args, flag)
		}
	}

	// Model
	add("--model", rc.ModelPath)

	// Server vs CLI mode
	if cfg.Mode == "server" {
		add("--host", cfg.Server.Host)
		add("--port", strconv.Itoa(cfg.Server.Port))
		if w.PublicPath != "" {
			add("--public", w.PublicPath)
		}
		if w.RequestPath != "" {
			add("--request-path", w.RequestPath)
		}
		if w.InferencePath != "" {
			add("--inference-path", w.InferencePath)
		}
		addIf("--convert", w.Convert)
		if w.TmpDir != "" {
			add("--tmp-dir", w.TmpDir)
		}
		addIf("--print-realtime", w.PrintRealtime)
		addIf("--no-language-probabilities", w.NoLanguageProbs)
	} else {
		// CLI output formats
		addIf("--output-txt", w.OutputTXT)
		addIf("--output-vtt", w.OutputVTT)
		addIf("--output-srt", w.OutputSRT)
		addIf("--output-lrc", w.OutputLRC)
		addIf("--output-words", w.OutputWords)
		addIf("--output-csv", w.OutputCSV)
		addIf("--output-json", w.OutputJSON)
		addIf("--output-json-full", w.OutputJSONFull)
		if w.OutputFile != "" {
			add("--output-file", w.OutputFile)
		}
		if w.FontPath != "" {
			add("--font-path", w.FontPath)
		}
		addIf("--no-prints", w.NoPrints)
		addIf("--print-confidence", w.PrintConfidence)
		addIf("--log-score", w.LogScore)
		if w.Temperature != 0 {
			add("--temperature", fmt.Sprintf("%.2f", w.Temperature))
		}
		if w.TemperatureInc != 0 {
			add("--temperature-inc", fmt.Sprintf("%.2f", w.TemperatureInc))
		}
		addIf("--carry-initial-prompt", w.CarryInitialPrompt)
		if w.SuppressRegex != "" {
			add("--suppress-regex", w.SuppressRegex)
		}
		if w.Grammar != "" {
			add("--grammar", w.Grammar)
		}
		if w.GrammarRule != "" {
			add("--grammar-rule", w.GrammarRule)
		}
		if w.GrammarPenalty > 0 && w.GrammarPenalty != 100.0 {
			add("--grammar-penalty", fmt.Sprintf("%.2f", w.GrammarPenalty))
		}
	}

	// Threads
	if p.Threads > 0 {
		add("--threads", strconv.Itoa(p.Threads))
	}
	if w.Processors > 0 {
		add("--processors", strconv.Itoa(w.Processors))
	}

	// Timing & segmentation
	if w.OffsetT > 0 {
		add("--offset-t", strconv.Itoa(w.OffsetT))
	}
	if w.OffsetN > 0 {
		add("--offset-n", strconv.Itoa(w.OffsetN))
	}
	if w.Duration > 0 {
		add("--duration", strconv.Itoa(w.Duration))
	}
	if w.MaxContext != 0 {
		add("--max-context", strconv.Itoa(w.MaxContext))
	}
	if w.MaxLen > 0 {
		add("--max-len", strconv.Itoa(w.MaxLen))
	}
	if w.AudioCtx > 0 {
		add("--audio-ctx", strconv.Itoa(w.AudioCtx))
	}
	addIf("--split-on-word", w.SplitOnWord)

	// Decoder thresholds
	if w.WordThreshold > 0 && w.WordThreshold != 0.01 {
		add("--word-thold", fmt.Sprintf("%.2f", w.WordThreshold))
	}
	if w.EntropyThreshold > 0 && w.EntropyThreshold != 2.40 {
		add("--entropy-thold", fmt.Sprintf("%.2f", w.EntropyThreshold))
	}
	if w.LogProbThreshold != 0 && w.LogProbThreshold != -1.0 {
		add("--logprob-thold", fmt.Sprintf("%.2f", w.LogProbThreshold))
	}
	if w.NoSpeechThreshold > 0 && w.NoSpeechThreshold != 0.60 {
		add("--no-speech-thold", fmt.Sprintf("%.2f", w.NoSpeechThreshold))
	}

	// Sampling
	if w.BeamSize > 0 {
		add("--beam-size", strconv.Itoa(w.BeamSize))
	}
	if w.BestOf > 0 {
		add("--best-of", strconv.Itoa(w.BestOf))
	}
	addIf("--no-fallback", w.NoFallback)

	// Language
	if w.Language != "" && w.Language != "auto" {
		add("--language", w.Language)
	}
	addIf("--detect-language", w.DetectLanguage)
	if w.Prompt != "" {
		add("--prompt", w.Prompt)
	}
	if w.Task == "translate" {
		args = append(args, "--translate")
	}

	// Diarization
	addIf("--diarize", w.Diarize)
	addIf("--tinydiarize", w.TinyDiarize)

	// Word-level timestamps (DTW)
	dtw := w.DTW
	if dtw == "" && w.WordTimestamps {
		dtw = "tiny"
	}
	if dtw != "" {
		add("--dtw", dtw)
	}

	// GPU
	addIf("--no-gpu", w.NoGPU)
	if w.Device > 0 {
		add("--device", strconv.Itoa(w.Device))
	}
	if w.FlashAttention != nil && !*w.FlashAttention {
		args = append(args, "--no-flash-attn")
	}

	// Suppress
	addIf("--suppress-nst", w.SuppressNST)

	// OpenVINO
	if w.OVEDevice != "" && w.OVEDevice != "CPU" {
		add("--ov-e-device", w.OVEDevice)
	}

	// Logging
	addIf("--debug-mode", w.DebugMode)
	addIf("--no-timestamps", w.NoTimestamps)
	addIf("--print-special", w.PrintSpecial)
	addIf("--print-colors", w.PrintColors)
	addIf("--print-progress", w.PrintProgress)

	// VAD
	if w.VAD {
		args = append(args, "--vad")
		if w.VADModel != "" {
			add("--vad-model", w.VADModel)
		}
		if w.VADThreshold > 0 && w.VADThreshold != 0.50 {
			add("--vad-threshold", fmt.Sprintf("%.2f", w.VADThreshold))
		}
		if w.VADMinSpeechDuration > 0 && w.VADMinSpeechDuration != 250 {
			add("--vad-min-speech-duration-ms", strconv.Itoa(w.VADMinSpeechDuration))
		}
		if w.VADMinSilenceDuration > 0 && w.VADMinSilenceDuration != 100 {
			add("--vad-min-silence-duration-ms", strconv.Itoa(w.VADMinSilenceDuration))
		}
		if w.VADMaxSpeechDuration > 0 {
			add("--vad-max-speech-duration-s", fmt.Sprintf("%.2f", w.VADMaxSpeechDuration))
		}
		if w.VADSpeechPad > 0 && w.VADSpeechPad != 30 {
			add("--vad-speech-pad-ms", strconv.Itoa(w.VADSpeechPad))
		}
		if w.VADSamplesOverlap > 0 && w.VADSamplesOverlap != 0.10 {
			add("--vad-samples-overlap", fmt.Sprintf("%.2f", w.VADSamplesOverlap))
		}
	}

	return args
}
