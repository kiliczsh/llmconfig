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
	// Core
	Language   string `yaml:"language"`   // "auto" | "en" | "tr" | ... (default: en)
	Task       string `yaml:"task"`       // "transcribe" | "translate"
	Processors int    `yaml:"processors"` // parallel processor count (default: 1)

	// Timing & segmentation
	OffsetT     int  `yaml:"offset_t"`      // time offset in milliseconds
	OffsetN     int  `yaml:"offset_n"`      // segment index offset
	Duration    int  `yaml:"duration"`      // audio duration to process in ms (0 = all)
	MaxContext  int  `yaml:"max_context"`   // max text context tokens (-1 = all)
	MaxLen      int  `yaml:"max_len"`       // max segment length in characters
	AudioCtx    int  `yaml:"audio_ctx"`     // audio context size (0 = all)
	SplitOnWord bool `yaml:"split_on_word"` // split on word rather than token

	// Decoder thresholds
	WordThreshold     float64 `yaml:"word_thold"`      // word timestamp probability threshold (default: 0.01)
	EntropyThreshold  float64 `yaml:"entropy_thold"`   // entropy threshold for decoder fail (default: 2.40)
	LogProbThreshold  float64 `yaml:"logprob_thold"`   // log probability threshold (default: -1.0)
	NoSpeechThreshold float64 `yaml:"no_speech_thold"` // no speech threshold (default: 0.60)

	// Sampling
	BeamSize       int     `yaml:"beam_size"`       // beam size for beam search (default: 5)
	BestOf         int     `yaml:"best_of"`         // best candidates to keep (default: 5)
	Temperature    float64 `yaml:"temperature"`     // sampling temperature (default: 0.0)
	TemperatureInc float64 `yaml:"temperature_inc"` // temperature increment (default: 0.20)
	NoFallback     bool    `yaml:"no_fallback"`     // no temperature fallback

	// Language
	DetectLanguage     bool   `yaml:"detect_language"`      // exit after auto-detecting language
	Prompt             string `yaml:"prompt"`               // initial prompt
	CarryInitialPrompt bool   `yaml:"carry_initial_prompt"` // always prepend initial prompt (cli)

	// Diarization
	Diarize     bool `yaml:"diarize"`     // stereo audio diarization
	TinyDiarize bool `yaml:"tinydiarize"` // tdrz model diarization

	// Word-level timestamps
	DTW string `yaml:"dtw"` // DTW model: tiny|base|small|medium|large-v1/v2/v3

	// GPU
	NoGPU          bool  `yaml:"no_gpu"`          // disable GPU
	Device         int   `yaml:"device"`          // GPU device ID (default: 0)
	FlashAttention *bool `yaml:"flash_attention"` // nil=default(on), false=--no-flash-attn

	// Suppress
	SuppressNST   bool   `yaml:"suppress_nst"`   // suppress non-speech tokens
	SuppressRegex string `yaml:"suppress_regex"` // regex matching tokens to suppress (cli)

	// Grammar (cli only)
	Grammar        string  `yaml:"grammar"`         // GBNF grammar to guide decoding
	GrammarRule    string  `yaml:"grammar_rule"`    // top-level grammar rule name
	GrammarPenalty float64 `yaml:"grammar_penalty"` // grammar penalty scale (default: 100.0)

	// OpenVINO
	OVEDevice string `yaml:"ov_e_device"` // OpenVINO encode device (default: CPU)

	// Output formats (cli only)
	OutputTXT      bool   `yaml:"output_txt"`       // output .txt file
	OutputVTT      bool   `yaml:"output_vtt"`       // output .vtt subtitle file
	OutputSRT      bool   `yaml:"output_srt"`       // output .srt subtitle file
	OutputLRC      bool   `yaml:"output_lrc"`       // output .lrc lyrics file
	OutputWords    bool   `yaml:"output_words"`     // output karaoke video script
	OutputCSV      bool   `yaml:"output_csv"`       // output .csv file
	OutputJSON     bool   `yaml:"output_json"`      // output .json file
	OutputJSONFull bool   `yaml:"output_json_full"` // include more info in JSON
	OutputFile     string `yaml:"output_file"`      // output file path (without extension)
	FontPath       string `yaml:"font_path"`        // monospace font for karaoke video
	NoTimestamps   bool   `yaml:"no_timestamps"`    // do not print timestamps

	// Logging / printing
	NoPrints        bool `yaml:"no_prints"`        // suppress all output except results (cli)
	PrintSpecial    bool `yaml:"print_special"`    // print special tokens
	PrintColors     bool `yaml:"print_colors"`     // print colors
	PrintConfidence bool `yaml:"print_confidence"` // print confidence (cli)
	PrintProgress   bool `yaml:"print_progress"`   // print progress
	PrintRealtime   bool `yaml:"print_realtime"`   // print output in realtime (server)
	LogScore        bool `yaml:"log_score"`        // log best decoder scores (cli)
	DebugMode       bool `yaml:"debug_mode"`       // debug mode (dump log_mel)

	// VAD (Voice Activity Detection)
	VAD                   bool    `yaml:"vad"`
	VADModel              string  `yaml:"vad_model"`
	VADThreshold          float64 `yaml:"vad_threshold"`               // speech threshold (default: 0.50)
	VADMinSpeechDuration  int     `yaml:"vad_min_speech_duration_ms"`  // min speech duration ms (default: 250)
	VADMinSilenceDuration int     `yaml:"vad_min_silence_duration_ms"` // min silence ms (default: 100)
	VADMaxSpeechDuration  float64 `yaml:"vad_max_speech_duration_s"`   // max speech duration s (default: unlimited)
	VADSpeechPad          int     `yaml:"vad_speech_pad_ms"`           // speech padding ms (default: 30)
	VADSamplesOverlap     float64 `yaml:"vad_samples_overlap"`         // samples overlap seconds (default: 0.10)

	// Server-only
	PublicPath      string `yaml:"public_path"`               // path to public folder
	RequestPath     string `yaml:"request_path"`              // request path prefix
	InferencePath   string `yaml:"inference_path"`            // inference path (default: /inference)
	Convert         bool   `yaml:"convert"`                   // convert audio to WAV via ffmpeg
	TmpDir          string `yaml:"tmp_dir"`                   // temp dir for ffmpeg files
	NoLanguageProbs bool   `yaml:"no_language_probabilities"` // exclude language probs from JSON output
}

type SDSpec struct {
	// Output (cli only)
	Output          string `yaml:"output"`           // output path, supports %d for sequences (default: ./output.png)
	PreviewPath     string `yaml:"preview_path"`     // preview image path
	PreviewInterval int    `yaml:"preview_interval"` // preview update interval in steps (default: 1)
	Preview         string `yaml:"preview"`          // preview method: none|proj|tae|vae
	Mode            string `yaml:"mode"`             // run mode: img_gen|vid_gen|upscale|convert|metadata

	// Server (server only)
	ServeHTMLPath string `yaml:"serve_html_path"` // path to HTML file to serve at root

	// Model components
	ClipL                   string `yaml:"clip_l"`
	ClipG                   string `yaml:"clip_g"`
	ClipVision              string `yaml:"clip_vision"`
	T5XXL                   string `yaml:"t5xxl"`
	LLM                     string `yaml:"llm"`
	LLMVision               string `yaml:"llm_vision"`
	DiffusionModel          string `yaml:"diffusion_model"`
	HighNoiseDiffusionModel string `yaml:"high_noise_diffusion_model"`
	VAE                     string `yaml:"vae"`
	TAESD                   string `yaml:"taesd"`
	ControlNet              string `yaml:"control_net"`
	EmbedDir                string `yaml:"embd_dir"`
	LoRAModelDir            string `yaml:"lora_model_dir"`
	TensorTypeRules         string `yaml:"tensor_type_rules"` // e.g. "^vae\.=f16,model\.=q8_0"
	PhotoMaker              string `yaml:"photo_maker"`
	UpscaleModel            string `yaml:"upscale_model"`

	// Hardware & weight type
	Type                string `yaml:"type"`                  // weight type: f32|f16|q4_0|q8_0|...
	RNG                 string `yaml:"rng"`                   // RNG: std_default|cuda|cpu
	SamplerRNG          string `yaml:"sampler_rng"`           // sampler RNG (default: use --rng)
	Prediction          string `yaml:"prediction"`            // prediction type: eps|v|edm_v|sd3_flow|flux_flow|flux2_flow
	LoRAApplyMode       string `yaml:"lora_apply_mode"`       // auto|immediately|at_runtime
	OffloadToCPU        bool   `yaml:"offload_to_cpu"`        // offload weights to RAM, load to VRAM when needed
	MMap                bool   `yaml:"mmap"`                  // memory-map model
	ControlNetCPU       bool   `yaml:"control_net_cpu"`       // keep controlnet in CPU
	ClipOnCPU           bool   `yaml:"clip_on_cpu"`           // keep clip in CPU
	VAEOnCPU            bool   `yaml:"vae_on_cpu"`            // keep vae in CPU
	FlashAttention      bool   `yaml:"flash_attention"`       // --fa: enable flash attention
	DiffusionFA         bool   `yaml:"diffusion_fa"`          // flash attention in diffusion model only
	DiffusionConvDirect bool   `yaml:"diffusion_conv_direct"` // ggml_conv2d_direct in diffusion
	VAEConvDirect       bool   `yaml:"vae_conv_direct"`       // ggml_conv2d_direct in VAE
	Circular            bool   `yaml:"circular"`              // circular padding for convolutions
	CircularX           bool   `yaml:"circular_x"`            // circular RoPE on x-axis (width)
	CircularY           bool   `yaml:"circular_y"`            // circular RoPE on y-axis (height)

	// Generation defaults
	Width                   int     `yaml:"width"`                      // image width in pixels (default: 512)
	Height                  int     `yaml:"height"`                     // image height in pixels (default: 512)
	Steps                   int     `yaml:"steps"`                      // sampling steps (default: 20)
	HighNoiseSteps          int     `yaml:"high_noise_steps"`           // high noise steps (-1 = auto)
	ClipSkip                int     `yaml:"clip_skip"`                  // CLIP layers to skip (-1 = auto)
	BatchCount              int     `yaml:"batch_count"`                // batch count
	VideoFrames             int     `yaml:"video_frames"`               // video frames (default: 1)
	FPS                     int     `yaml:"fps"`                        // FPS for video (default: 24)
	TimestepShift           int     `yaml:"timestep_shift"`             // timestep shift for NitroFusion
	UpscaleRepeats          int     `yaml:"upscale_repeats"`            // ESRGAN upscale repeats (default: 1)
	UpscaleTileSize         int     `yaml:"upscale_tile_size"`          // ESRGAN tile size (default: 128)
	CFGScale                float64 `yaml:"cfg_scale"`                  // guidance scale (default: 7.0)
	ImgCFGScale             float64 `yaml:"img_cfg_scale"`              // image guidance scale for inpaint
	Guidance                float64 `yaml:"guidance"`                   // distilled guidance scale (default: 3.5)
	SLGScale                float64 `yaml:"slg_scale"`                  // skip layer guidance scale (0 = disabled)
	SkipLayerStart          float64 `yaml:"skip_layer_start"`           // SLG enabling point (default: 0.01)
	SkipLayerEnd            float64 `yaml:"skip_layer_end"`             // SLG disabling point (default: 0.2)
	Eta                     float64 `yaml:"eta"`                        // noise multiplier
	FlowShift               float64 `yaml:"flow_shift"`                 // flow shift for SD3/WAN (0 = auto)
	Strength                float64 `yaml:"strength"`                   // noise strength for img2img (default: 0.75)
	ControlStrength         float64 `yaml:"control_strength"`           // control net strength (default: 0.9)
	VAETileOverlap          float64 `yaml:"vae_tile_overlap"`           // VAE tile overlap fraction (default: 0.5)
	Seed                    int64   `yaml:"seed"`                       // RNG seed (default: 42, <0 = random)
	SamplingMethod          string  `yaml:"sampling_method"`            // euler|euler_a|heun|dpm++2m|dpm++2s_a|...
	HighNoiseSamplingMethod string  `yaml:"high_noise_sampling_method"` // sampling method for high noise stage
	Scheduler               string  `yaml:"scheduler"`                  // sigma scheduler: discrete|karras|exponential|ays|...
	NegativePrompt          string  `yaml:"negative_prompt"`            // default negative prompt
	VAETiling               bool    `yaml:"vae_tiling"`                 // process VAE in tiles to reduce memory
	VAETileSize             string  `yaml:"vae_tile_size"`              // VAE tile size, format: "32x32"
	VAERelativeTileSize     string  `yaml:"vae_relative_tile_size"`     // relative VAE tile size
	DisableImageMetadata    bool    `yaml:"disable_image_metadata"`     // do not embed generation metadata
	SkipLayers              string  `yaml:"skip_layers"`                // SLG skip layers, e.g. "[7,8,9]"
	Sigmas                  string  `yaml:"sigmas"`                     // custom sigma values, comma-separated
	RefImage                string  `yaml:"ref_image"`                  // reference image for Flux Kontext

	// High noise stage (two-stage generation)
	HighNoiseCFGScale       float64 `yaml:"high_noise_cfg_scale"`
	HighNoiseImgCFGScale    float64 `yaml:"high_noise_img_cfg_scale"`
	HighNoiseGuidance       float64 `yaml:"high_noise_guidance"`
	HighNoiseSLGScale       float64 `yaml:"high_noise_slg_scale"`
	HighNoiseSkipLayerStart float64 `yaml:"high_noise_skip_layer_start"`
	HighNoiseSkipLayerEnd   float64 `yaml:"high_noise_skip_layer_end"`
	HighNoiseEta            float64 `yaml:"high_noise_eta"`
	HighNoiseSkipLayers     string  `yaml:"high_noise_skip_layers"`

	// Cache acceleration
	CacheMode   string `yaml:"cache_mode"`   // easycache|ucache|dbcache|taylorseer|cache-dit|spectrum
	CacheOption string `yaml:"cache_option"` // key=value cache params, comma-separated
	SCMMask     string `yaml:"scm_mask"`     // SCM steps mask, e.g. "1,1,1,0,0,1"
	SCMPolicy   string `yaml:"scm_policy"`   // SCM policy: dynamic|static

	// Logging
	Verbose bool `yaml:"verbose"` // print extra info
	Color   bool `yaml:"color"`   // colored logging
}
