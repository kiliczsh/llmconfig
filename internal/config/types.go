package config

type Config struct {
	Version          int              `yaml:"version"`
	Name             string           `yaml:"name"`
	Description      string           `yaml:"description"`
	Tags             []string         `yaml:"tags"`
	Meta             Meta             `yaml:"meta"`
	Backend          string           `yaml:"backend"` // "llama" | "sd" | "whisper" (default: "llama")
	Model            ModelSpec        `yaml:"model"`
	Mode             string           `yaml:"mode"`
	Server           ServerSpec       `yaml:"server"`
	HardwareProfiles HardwareProfiles `yaml:"hardware_profiles"`
	Context          ContextSpec      `yaml:"context"`
	Sampling         SamplingSpec     `yaml:"sampling"`
	Chat             ChatSpec         `yaml:"chat"`
	Rope             RopeSpec         `yaml:"rope"`
	Resources        ResourceSpec     `yaml:"resources"`
	Logging          LoggingSpec      `yaml:"logging"`
	Whisper          WhisperSpec      `yaml:"whisper,omitempty"`
	SD               SDSpec           `yaml:"sd,omitempty"`

	// internal: resolved file path
	FilePath string `yaml:"-"`
}

type Meta struct {
	Author    string `yaml:"author"`
	CreatedAt string `yaml:"created_at"`
	UpdatedAt string `yaml:"updated_at"`
	Notes     string `yaml:"notes"`
}

type ModelSpec struct {
	Source                  string       `yaml:"source"`
	Repo                    string       `yaml:"repo"`
	File                    string       `yaml:"file"`
	Path                    string       `yaml:"path"`
	URL                     string       `yaml:"url"`
	Checksum                string       `yaml:"checksum"`
	Download                DownloadSpec `yaml:"download"`
	Draft                   *DraftSpec   `yaml:"draft,omitempty"`
	MMProj                  *MMProjSpec  `yaml:"mmproj,omitempty"`
	LoRA                    []string     `yaml:"lora"`                       // LoRA adapter file(s)
	LoRAScaled              []string     `yaml:"lora_scaled"`                // LoRA with scaling, format: "FNAME:SCALE"
	ControlVector           []string     `yaml:"control_vector"`             // control vector file(s)
	ControlVectorScaled     []string     `yaml:"control_vector_scaled"`      // control vector with scaling, format: "FNAME:SCALE"
	ControlVectorLayerStart int          `yaml:"control_vector_layer_start"` // layer range start (-1 = not set)
	ControlVectorLayerEnd   int          `yaml:"control_vector_layer_end"`   // layer range end (-1 = not set)
	OverrideKV              []string     `yaml:"override_kv"`                // model metadata overrides, format: "KEY=TYPE:VALUE"
}

type DownloadSpec struct {
	// Pointer so an explicit `false` in YAML is honored; ApplyDefaults fills nil with true.
	VerifyChecksum *bool  `yaml:"verify_checksum"`
	Resume         *bool  `yaml:"resume"`
	Connections    int    `yaml:"connections"`
	CacheDir       string `yaml:"cache_dir"`
}

type DraftSpec struct {
	Source            string   `yaml:"source"`
	Repo              string   `yaml:"repo"`
	File              string   `yaml:"file"`
	DraftN            int      `yaml:"draft_n"`
	DraftMin          int      `yaml:"draft_min"`           // minimum draft tokens (default: 0)
	DraftPMin         float64  `yaml:"draft_p_min"`         // minimum draft probability (default: 0.75)
	NCtx              int      `yaml:"n_ctx"`               // draft model context size
	NGPULayers        int      `yaml:"n_gpu_layers"`        // draft model GPU layers
	Devices           []string `yaml:"devices"`             // draft model GPU devices
	CacheTypeK        string   `yaml:"cache_type_k"`        // draft KV cache type K
	CacheTypeV        string   `yaml:"cache_type_v"`        // draft KV cache type V
	SpecReplaceTarget string   `yaml:"spec_replace_target"` // speculative decoding: target string
	SpecReplaceDraft  string   `yaml:"spec_replace_draft"`  // speculative decoding: draft replacement
	OverrideTensor    []string `yaml:"override_tensor"`     // tensor buffer overrides for draft model
	CPUMoE            bool     `yaml:"cpu_moe"`             // keep all MoE weights in CPU for draft
	NCPUMoE           int      `yaml:"n_cpu_moe"`           // keep first N layers MoE in CPU for draft
	ThreadsDraft      int      `yaml:"threads_draft"`       // draft model generation threads (server only)
	ThreadsBatchDraft int      `yaml:"threads_batch_draft"` // draft model batch threads (server only)
	SpecType          string   `yaml:"spec_type"`           // speculative decoding type: none|ngram-cache|ngram-simple|...
	SpecNgramSizeN    int      `yaml:"spec_ngram_size_n"`   // ngram lookup size N (default: 12)
	SpecNgramSizeM    int      `yaml:"spec_ngram_size_m"`   // ngram draft size M (default: 48)
	SpecNgramMinHits  int      `yaml:"spec_ngram_min_hits"` // min hits for ngram-map (default: 1)
}

type MMProjSpec struct {
	Source  string `yaml:"source"`
	Repo    string `yaml:"repo"`
	File    string `yaml:"file"`
	Offload *bool  `yaml:"offload"` // GPU offload for mmproj (default: enabled)
}

type ServerSpec struct {
	Host         string       `yaml:"host"`
	Port         int          `yaml:"port"`
	APIKey       string       `yaml:"api_key"`
	APIKeyFile   string       `yaml:"api_key_file"` // path to file containing API keys
	CORSOrigins  []string     `yaml:"cors_origins"`
	Parallel     int          `yaml:"parallel"`
	QueueSize    int          `yaml:"queue_size"`
	Endpoints    EndpointSpec `yaml:"endpoints"`
	ReadTimeout  string       `yaml:"read_timeout"`
	WriteTimeout string       `yaml:"write_timeout"`
	Timeout      int          `yaml:"timeout"`      // server read/write timeout in seconds (0 = not set)
	ThreadsHTTP  int          `yaml:"threads_http"` // HTTP request handler threads (-1 = auto)
	ReusePort    bool         `yaml:"reuse_port"`   // allow multiple sockets on same port
	StaticPath   string       `yaml:"path"`         // path to serve static files
	APIPrefix    string       `yaml:"api_prefix"`   // API path prefix (without trailing slash)

	// SSL
	SSLKeyFile  string `yaml:"ssl_key_file"`  // PEM-encoded SSL private key
	SSLCertFile string `yaml:"ssl_cert_file"` // PEM-encoded SSL certificate

	// Prompt & cache
	CachePrompt *bool `yaml:"cache_prompt"` // prompt caching (default: enabled)
	CacheReuse  int   `yaml:"cache_reuse"`  // min chunk size for KV-shift cache reuse (0 = disabled)

	// Slots
	SlotPromptSimilarity float64 `yaml:"slot_prompt_similarity"`  // slot matching threshold (0.0 = disabled, default: 0.10)
	SlotSavePath         string  `yaml:"slot_save_path"`          // path to save slot KV cache
	SleepIdleSeconds     int     `yaml:"sleep_idle_seconds"`      // sleep after idle (-1 = disabled, 0 = not set)
	LoRAInitWithoutApply bool    `yaml:"lora_init_without_apply"` // load LoRA without applying

	// KV
	KVUnified *bool `yaml:"kv_unified"` // unified KV buffer (default: auto)
	ClearIdle *bool `yaml:"clear_idle"` // save and clear idle slots on new task

	// Continuous batching
	ContBatching *bool `yaml:"cont_batching"` // continuous batching (default: enabled)

	// Model identification
	Alias string `yaml:"alias"` // model name aliases, comma-separated
	Tags  string `yaml:"tags"`  // model tags, comma-separated

	// Web UI
	WebUI           *bool  `yaml:"webui"`             // enable/disable Web UI (default: enabled)
	WebUIConfig     string `yaml:"webui_config"`      // JSON for default WebUI settings
	WebUIConfigFile string `yaml:"webui_config_file"` // path to JSON file for WebUI settings
	WebUIMCPProxy   bool   `yaml:"webui_mcp_proxy"`   // experimental: MCP CORS proxy

	// Agent tools (experimental)
	Tools string `yaml:"tools"` // comma-separated built-in tools, or "all"

	// Embeddings & reranking
	Pooling string `yaml:"pooling"` // pooling type: none|mean|cls|last|rank

	// Infill
	SPMInfill bool `yaml:"spm_infill"` // use Suffix/Prefix/Middle pattern (default: Prefix/Suffix/Middle)

	// Prefill
	PrefillAssistant *bool `yaml:"prefill_assistant"` // prefill assistant response (default: enabled)

	// Media
	MediaPath string `yaml:"media_path"` // directory for local media files (file:// URLs)

	// Router server
	ModelsDir      string `yaml:"models_dir"`      // directory of models for router server
	ModelsPreset   string `yaml:"models_preset"`   // INI file for model presets
	ModelsMax      int    `yaml:"models_max"`      // max simultaneous models (-1 = not set, 0 = unlimited)
	ModelsAutoload *bool  `yaml:"models_autoload"` // auto-load models (default: enabled)

	// Lookup cache (for lookup decoding)
	LookupCacheStatic  string `yaml:"lookup_cache_static"`  // path to static lookup cache
	LookupCacheDynamic string `yaml:"lookup_cache_dynamic"` // path to dynamic lookup cache
}

type EndpointSpec struct {
	Metrics bool `yaml:"metrics"`
	// Pointer so an explicit `false` in YAML is honored; ApplyDefaults fills nil with true.
	Slots       *bool `yaml:"slots"`
	Health      bool  `yaml:"health"`
	Completions bool  `yaml:"completions"`
	Chat        bool  `yaml:"chat"`
	Embeddings  bool  `yaml:"embeddings"`
	Rerank      bool  `yaml:"rerank"` // reranking endpoint
	Props       bool  `yaml:"props"`  // POST /props endpoint for dynamic property changes
}

type HardwareProfiles struct {
	AppleSilicon HardwareProfile `yaml:"apple_silicon"`
	NVIDIA       HardwareProfile `yaml:"nvidia"`
	AMD          HardwareProfile `yaml:"amd"`
	IntelGPU     HardwareProfile `yaml:"intel_gpu"`
	CPU          HardwareProfile `yaml:"cpu"`
}

type HardwareProfile struct {
	NGPULayers     int       `yaml:"n_gpu_layers"`
	Metal          bool      `yaml:"metal"`
	CUDA           bool      `yaml:"cuda"`
	ROCm           bool      `yaml:"rocm"`
	SYCL           bool      `yaml:"sycl"`
	Threads        int       `yaml:"threads"`
	ThreadsBatch   int       `yaml:"threads_batch"`
	Devices        []string  `yaml:"devices"`
	TensorSplit    []float64 `yaml:"tensor_split"`
	CPURange       string    `yaml:"cpu_range"`
	CPUStrict      bool      `yaml:"cpu_strict"`
	NUMA           string    `yaml:"numa"`
	SplitMode      string    `yaml:"split_mode"`       // "none" | "layer" | "row" | "tensor"
	MainGPU        int       `yaml:"main_gpu"`         // main GPU index (-1 = use default)
	Priority       int       `yaml:"priority"`         // -1=low, 0=normal, 1=medium, 2=high, 3=realtime
	Fit            string    `yaml:"fit"`              // "on" | "off" — auto-fit to device memory
	FitTarget      []int     `yaml:"fit_target"`       // target margin per device in MiB
	FitCtx         int       `yaml:"fit_ctx"`          // minimum ctx size fit can set
	OverrideTensor []string  `yaml:"override_tensor"`  // tensor buffer overrides, format: "pattern=type"
	CPUMoE         bool      `yaml:"cpu_moe"`          // keep all MoE weights in CPU
	CPUMask        string    `yaml:"cpu_mask"`         // CPU affinity mask (hex)
	CPUMaskBatch   string    `yaml:"cpu_mask_batch"`   // CPU affinity mask for batch processing
	Poll           *int      `yaml:"poll"`             // polling level 0-100 (default: 50)
	PollBatch      *int      `yaml:"poll_batch"`       // polling level for batch (default: same as poll)
	PrioBatch      int       `yaml:"prio_batch"`       // batch thread priority: 0=normal,1=medium,2=high,3=realtime
	CPURangeBatch  string    `yaml:"cpu_range_batch"`  // CPU range for batch affinity (e.g. "0-3")
	CPUStrictBatch bool      `yaml:"cpu_strict_batch"` // strict CPU placement for batch
	Repack         *bool     `yaml:"repack"`           // weight repacking (default: enabled)
	NoHost         bool      `yaml:"no_host"`          // bypass host buffer for extra device buffers
	OpOffload      *bool     `yaml:"op_offload"`       // offload host tensor ops to device (default: enabled)
	RPC            string    `yaml:"rpc"`              // RPC servers, comma-separated host:port
	DirectIO       *bool     `yaml:"direct_io"`        // use DirectIO if available (default: disabled)
}

type ContextSpec struct {
	NCtx       int    `yaml:"n_ctx"`
	NBatch     int    `yaml:"n_batch"`
	NUBatch    int    `yaml:"n_ubatch"`
	NKeep      int    `yaml:"n_keep"`
	CacheTypeK string `yaml:"cache_type_k"`
	CacheTypeV string `yaml:"cache_type_v"`
	// Pointer so an explicit `false` in YAML is honored; ApplyDefaults fills nil with true.
	MMap                   *bool `yaml:"mmap"`
	MLock                  bool  `yaml:"mlock"`
	FlashAttention         bool  `yaml:"flash_attention"`
	NCPUMoE                int   `yaml:"n_cpu_moe"`
	NPredict               int   `yaml:"n_predict"`                 // tokens to predict (-1 = infinity, 0 = not set)
	ContextShift           bool  `yaml:"context_shift"`             // enable context shift on infinite generation
	KVOffload              *bool `yaml:"kv_offload"`                // KV cache GPU offload (default: enabled)
	SWAFull                bool  `yaml:"swa_full"`                  // use full-size SWA cache
	CacheRAM               int   `yaml:"cache_ram"`                 // max RAM cache in MiB (-1 = no limit, 0 = not set)
	ImageMinTokens         int   `yaml:"image_min_tokens"`          // min tokens per image (vision models)
	ImageMaxTokens         int   `yaml:"image_max_tokens"`          // max tokens per image (vision models)
	CheckTensors           bool  `yaml:"check_tensors"`             // validate tensor data on load
	CtxCheckpoints         int   `yaml:"ctx_checkpoints"`           // max context checkpoints per slot
	CheckpointEveryNTokens int   `yaml:"checkpoint_every_n_tokens"` // checkpoint interval (-1 = disable, 0 = not set)
}

type SamplingSpec struct {
	Temperature         float64  `yaml:"temperature"`
	TopK                int      `yaml:"top_k"`
	TopP                float64  `yaml:"top_p"`
	MinP                float64  `yaml:"min_p"`
	RepeatPenalty       float64  `yaml:"repeat_penalty"`
	RepeatLastN         int      `yaml:"repeat_last_n"`
	PresencePenalty     float64  `yaml:"presence_penalty"`  // presence penalty (0.0 = disabled)
	FrequencyPenalty    float64  `yaml:"frequency_penalty"` // frequency penalty (0.0 = disabled)
	DryMultiplier       float64  `yaml:"dry_multiplier"`
	DryBase             float64  `yaml:"dry_base"`
	DryAllowedLength    int      `yaml:"dry_allowed_length"`
	DryPenaltyLastN     int      `yaml:"dry_penalty_last_n"`
	DynatempRange       float64  `yaml:"dynatemp_range"`  // dynamic temperature range (0.0 = disabled)
	DynatempExp         float64  `yaml:"dynatemp_exp"`    // dynamic temperature exponent
	XTCProbability      float64  `yaml:"xtc_probability"` // XTC probability (0.0 = disabled)
	XTCThreshold        float64  `yaml:"xtc_threshold"`   // XTC threshold
	Mirostat            int      `yaml:"mirostat"`
	MirostatTau         float64  `yaml:"mirostat_tau"`
	MirostatEta         float64  `yaml:"mirostat_eta"`
	Samplers            string   `yaml:"samplers"`
	Seed                int64    `yaml:"seed"`                  // RNG seed (0 = not set, use random)
	Typical             float64  `yaml:"typical"`               // locally typical sampling (0 = not set, 1.0 = disabled)
	TopNSigma           float64  `yaml:"top_nsigma"`            // top-n-sigma sampling (-1 = disabled, 0 = not set)
	AdaptiveTarget      float64  `yaml:"adaptive_target"`       // adaptive-p target (-1 = disabled, 0 = not set)
	AdaptiveDecay       float64  `yaml:"adaptive_decay"`        // adaptive-p decay rate
	DrySequenceBreakers []string `yaml:"dry_sequence_breakers"` // custom DRY sequence breakers
	BackendSampling     bool     `yaml:"backend_sampling"`      // enable backend sampling (experimental)
	SamplerSeq          string   `yaml:"sampler_seq"`           // simplified sampler sequence (e.g. "edskypmxt")
	IgnoreEOS           bool     `yaml:"ignore_eos"`            // ignore end-of-stream token
	Grammar             string   `yaml:"grammar"`               // BNF-like grammar string
	GrammarFile         string   `yaml:"grammar_file"`          // path to grammar file
	JSONSchema          string   `yaml:"json_schema"`           // JSON schema string
	JSONSchemaFile      string   `yaml:"json_schema_file"`      // path to JSON schema file
}

type ChatSpec struct {
	Template               string            `yaml:"template"`
	SystemPrompt           string            `yaml:"system_prompt"`
	TemplateKwargs         map[string]string `yaml:"template_kwargs"`
	Jinja                  bool              `yaml:"jinja"`
	Reasoning              string            `yaml:"reasoning"`                // "on" | "off" | "auto"
	ReasoningBudget        *int              `yaml:"reasoning_budget"`         // -1 unlimited, 0 immediate end, N>0 budget
	ReasoningBudgetMessage string            `yaml:"reasoning_budget_message"` // message injected when budget exhausted
	ReasoningFormat        string            `yaml:"reasoning_format"`         // "none" | "deepseek" | "deepseek-legacy"
	TemplateFile           string            `yaml:"template_file"`            // path to jinja template file
	SkipChatParsing        bool              `yaml:"skip_chat_parsing"`        // force pure content parser
}

type RopeSpec struct {
	Scaling        string  `yaml:"scaling"`
	Scale          float64 `yaml:"scale"` // context scaling factor (--rope-scale)
	FreqBase       float64 `yaml:"freq_base"`
	FreqScale      float64 `yaml:"freq_scale"`
	YarnExtFactor  float64 `yaml:"yarn_ext_factor"`
	YarnAttnFactor float64 `yaml:"yarn_attn_factor"`
	YarnBetaSlow   float64 `yaml:"yarn_beta_slow"` // YaRN high correction dim / alpha
	YarnBetaFast   float64 `yaml:"yarn_beta_fast"` // YaRN low correction dim / beta
	YarnOrigCtx    int     `yaml:"yarn_orig_ctx"`
}

type ResourceSpec struct {
	VRAMLimit   string `yaml:"vram_limit"`
	VRAMBuffer  string `yaml:"vram_buffer"`
	MemoryLimit string `yaml:"memory_limit"`
	CPULimit    string `yaml:"cpu_limit"`
	CPUPriority string `yaml:"cpu_priority"`
	// Pointer so an explicit `false` in YAML is honored; ApplyDefaults fills nil with true.
	FallbackToCPU  *bool  `yaml:"fallback_to_cpu"`
	RequestTimeout string `yaml:"request_timeout"`
	MaxConcurrent  int    `yaml:"max_concurrent"`
}

type LoggingSpec struct {
	Level       string `yaml:"level"`
	File        string `yaml:"file"`
	Colors      string `yaml:"colors"`
	Prefix      bool   `yaml:"prefix"`       // enable prefix in log messages
	Timestamps  bool   `yaml:"timestamps"`   // enable timestamps in log messages
	Verbosity   int    `yaml:"verbosity"`    // verbosity threshold: 0=generic,1=error,2=warn,3=info,4=debug (-1 = not set)
	ShowTimings *bool  `yaml:"show_timings"` // show timing info after each response (default: true)
}

// RunConfig is the flattened, resolved configuration passed to the runner.
type RunConfig struct {
	Config         *Config
	ModelPath      string
	DraftModelPath string
	MMProjPath     string
	Profile        HardwareProfile
	ProfileName    string
	LogFile        string
	BinaryPath     string
	Backend        string // "llama" | "sd" | "whisper"
}

type WhisperSpec struct {
	Language       string  `yaml:"language"` // "auto" | "en" | "tr" | ...
	Task           string  `yaml:"task"`     // "transcribe" | "translate"
	BeamSize       int     `yaml:"beam_size"`
	BestOf         int     `yaml:"best_of"`
	VAD            bool    `yaml:"vad"`
	VADThreshold   float64 `yaml:"vad_threshold"`
	WordTimestamps bool    `yaml:"word_timestamps"`
	Processors     int     `yaml:"processors"`
}

type SDSpec struct {
	Width          int     `yaml:"width"`
	Height         int     `yaml:"height"`
	Steps          int     `yaml:"steps"`
	CFGScale       float64 `yaml:"cfg_scale"`
	SamplingMethod string  `yaml:"sampling_method"` // "euler_a" | "euler" | "dpm++2m" | ...
	Seed           int64   `yaml:"seed"`
}
