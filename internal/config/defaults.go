package config

func ApplyDefaults(cfg *Config) {
	if cfg.Backend == "" {
		cfg.Backend = "llama"
	}

	if cfg.Mode == "" {
		cfg.Mode = "server"
	}

	// Backend-specific defaults
	if cfg.Backend == "whisper" {
		if cfg.Whisper.Language == "" {
			cfg.Whisper.Language = "auto"
		}
		if cfg.Whisper.Task == "" {
			cfg.Whisper.Task = "transcribe"
		}
		if cfg.Whisper.BeamSize == 0 {
			cfg.Whisper.BeamSize = 5
		}
		if cfg.Whisper.BestOf == 0 {
			cfg.Whisper.BestOf = 5
		}
		if cfg.Whisper.VADThreshold == 0 {
			cfg.Whisper.VADThreshold = 0.5
		}
		if cfg.Whisper.Processors == 0 {
			cfg.Whisper.Processors = 1
		}
	}

	if cfg.Backend == "sd" {
		if cfg.SD.Width == 0 {
			cfg.SD.Width = 512
		}
		if cfg.SD.Height == 0 {
			cfg.SD.Height = 512
		}
		if cfg.SD.Steps == 0 {
			cfg.SD.Steps = 20
		}
		if cfg.SD.CFGScale == 0 {
			cfg.SD.CFGScale = 7.0
		}
		if cfg.SD.SamplingMethod == "" {
			cfg.SD.SamplingMethod = "euler_a"
		}
		if cfg.SD.Seed == 0 {
			cfg.SD.Seed = -1
		}
	}

	// Server defaults
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Parallel == 0 {
		cfg.Server.Parallel = 1
	}
	if cfg.Server.QueueSize == 0 {
		cfg.Server.QueueSize = 10
	}
	if cfg.Server.ReadTimeout == "" {
		cfg.Server.ReadTimeout = "600s"
	}
	if cfg.Server.WriteTimeout == "" {
		cfg.Server.WriteTimeout = "600s"
	}
	cfg.Server.Endpoints.Health = true
	cfg.Server.Endpoints.Slots = true
	cfg.Server.Endpoints.Completions = true
	cfg.Server.Endpoints.Chat = true

	// Download defaults
	if !cfg.Model.Download.VerifyChecksum {
		cfg.Model.Download.VerifyChecksum = true
	}
	if !cfg.Model.Download.Resume {
		cfg.Model.Download.Resume = true
	}
	if cfg.Model.Download.Connections == 0 {
		cfg.Model.Download.Connections = 4
	}

	// Context defaults
	if cfg.Context.NCtx == 0 {
		cfg.Context.NCtx = 4096
	}
	if cfg.Context.NBatch == 0 {
		cfg.Context.NBatch = 512
	}
	if cfg.Context.NUBatch == 0 {
		cfg.Context.NUBatch = cfg.Context.NBatch
	}
	if cfg.Context.CacheTypeK == "" {
		cfg.Context.CacheTypeK = "f16"
	}
	if cfg.Context.CacheTypeV == "" {
		cfg.Context.CacheTypeV = "f16"
	}
	if !cfg.Context.MMap {
		cfg.Context.MMap = true
	}

	// Sampling defaults
	if cfg.Sampling.Temperature == 0 {
		cfg.Sampling.Temperature = 0.8
	}
	if cfg.Sampling.TopK == 0 {
		cfg.Sampling.TopK = 40
	}
	if cfg.Sampling.TopP == 0 {
		cfg.Sampling.TopP = 0.95
	}
	if cfg.Sampling.MinP == 0 {
		cfg.Sampling.MinP = 0.05
	}
	if cfg.Sampling.RepeatPenalty == 0 {
		cfg.Sampling.RepeatPenalty = 1.0
	}
	if cfg.Sampling.RepeatLastN == 0 {
		cfg.Sampling.RepeatLastN = 64
	}
	if cfg.Sampling.DryBase == 0 {
		cfg.Sampling.DryBase = 1.75
	}
	if cfg.Sampling.DryAllowedLength == 0 {
		cfg.Sampling.DryAllowedLength = 2
	}
	if cfg.Sampling.DryPenaltyLastN == 0 {
		cfg.Sampling.DryPenaltyLastN = -1
	}
	if cfg.Sampling.MirostatTau == 0 {
		cfg.Sampling.MirostatTau = 5.0
	}
	if cfg.Sampling.MirostatEta == 0 {
		cfg.Sampling.MirostatEta = 0.1
	}

	// RoPE defaults
	if cfg.Rope.YarnExtFactor == 0 {
		cfg.Rope.YarnExtFactor = -1.0
	}
	if cfg.Rope.YarnAttnFactor == 0 {
		cfg.Rope.YarnAttnFactor = 1.0
	}

	// Resources defaults
	if cfg.Resources.VRAMBuffer == "" {
		cfg.Resources.VRAMBuffer = "512MB"
	}
	if cfg.Resources.CPUPriority == "" {
		cfg.Resources.CPUPriority = "normal"
	}
	if !cfg.Resources.FallbackToCPU {
		cfg.Resources.FallbackToCPU = true
	}
	if cfg.Resources.RequestTimeout == "" {
		cfg.Resources.RequestTimeout = "120s"
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Colors == "" {
		cfg.Logging.Colors = "auto"
	}

	// Draft defaults
	if cfg.Model.Draft != nil && cfg.Model.Draft.DraftN == 0 {
		cfg.Model.Draft.DraftN = 5
	}
}
