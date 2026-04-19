package config

type Config struct {
	Version          int              `yaml:"version"`
	Name             string           `yaml:"name"`
	Description      string           `yaml:"description"`
	Tags             []string         `yaml:"tags"`
	Meta             Meta             `yaml:"meta"`
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
	Source   string       `yaml:"source"`
	Repo     string       `yaml:"repo"`
	File     string       `yaml:"file"`
	Path     string       `yaml:"path"`
	URL      string       `yaml:"url"`
	Checksum string       `yaml:"checksum"`
	Download DownloadSpec `yaml:"download"`
	Draft    *DraftSpec   `yaml:"draft,omitempty"`
	MMProj   *MMProjSpec  `yaml:"mmproj,omitempty"`
}

type DownloadSpec struct {
	VerifyChecksum bool   `yaml:"verify_checksum"`
	Resume         bool   `yaml:"resume"`
	Connections    int    `yaml:"connections"`
	CacheDir       string `yaml:"cache_dir"`
}

type DraftSpec struct {
	Source string `yaml:"source"`
	Repo   string `yaml:"repo"`
	File   string `yaml:"file"`
	DraftN int    `yaml:"draft_n"`
}

type MMProjSpec struct {
	Source string `yaml:"source"`
	Repo   string `yaml:"repo"`
	File   string `yaml:"file"`
}

type ServerSpec struct {
	Host         string       `yaml:"host"`
	Port         int          `yaml:"port"`
	APIKey       string       `yaml:"api_key"`
	CORSOrigins  []string     `yaml:"cors_origins"`
	Parallel     int          `yaml:"parallel"`
	QueueSize    int          `yaml:"queue_size"`
	Endpoints    EndpointSpec `yaml:"endpoints"`
	ReadTimeout  string       `yaml:"read_timeout"`
	WriteTimeout string       `yaml:"write_timeout"`
}

type EndpointSpec struct {
	Metrics     bool `yaml:"metrics"`
	Slots       bool `yaml:"slots"`
	Health      bool `yaml:"health"`
	Completions bool `yaml:"completions"`
	Chat        bool `yaml:"chat"`
	Embeddings  bool `yaml:"embeddings"`
}

type HardwareProfiles struct {
	AppleSilicon HardwareProfile `yaml:"apple_silicon"`
	NVIDIA       HardwareProfile `yaml:"nvidia"`
	AMD          HardwareProfile `yaml:"amd"`
	IntelGPU     HardwareProfile `yaml:"intel_gpu"`
	CPU          HardwareProfile `yaml:"cpu"`
}

type HardwareProfile struct {
	NGPULayers   int       `yaml:"n_gpu_layers"`
	Metal        bool      `yaml:"metal"`
	CUDA         bool      `yaml:"cuda"`
	ROCm         bool      `yaml:"rocm"`
	SYCL         bool      `yaml:"sycl"`
	Threads      int       `yaml:"threads"`
	ThreadsBatch int       `yaml:"threads_batch"`
	Devices      []string  `yaml:"devices"`
	TensorSplit  []float64 `yaml:"tensor_split"`
	CPURange     string    `yaml:"cpu_range"`
	CPUStrict    bool      `yaml:"cpu_strict"`
	NUMA         string    `yaml:"numa"`
}

type ContextSpec struct {
	NCtx           int    `yaml:"n_ctx"`
	NBatch         int    `yaml:"n_batch"`
	NUBatch        int    `yaml:"n_ubatch"`
	NKeep          int    `yaml:"n_keep"`
	CacheTypeK     string `yaml:"cache_type_k"`
	CacheTypeV     string `yaml:"cache_type_v"`
	MMap           bool   `yaml:"mmap"`
	MLock          bool   `yaml:"mlock"`
	FlashAttention bool   `yaml:"flash_attention"`
	NCPUMoE        int    `yaml:"n_cpu_moe"`
}

type SamplingSpec struct {
	Temperature      float64 `yaml:"temperature"`
	TopK             int     `yaml:"top_k"`
	TopP             float64 `yaml:"top_p"`
	MinP             float64 `yaml:"min_p"`
	RepeatPenalty    float64 `yaml:"repeat_penalty"`
	RepeatLastN      int     `yaml:"repeat_last_n"`
	DryMultiplier    float64 `yaml:"dry_multiplier"`
	DryBase          float64 `yaml:"dry_base"`
	DryAllowedLength int     `yaml:"dry_allowed_length"`
	DryPenaltyLastN  int     `yaml:"dry_penalty_last_n"`
	Mirostat         int     `yaml:"mirostat"`
	MirostatTau      float64 `yaml:"mirostat_tau"`
	MirostatEta      float64 `yaml:"mirostat_eta"`
	Samplers         string  `yaml:"samplers"`
}

type ChatSpec struct {
	Template       string            `yaml:"template"`
	SystemPrompt   string            `yaml:"system_prompt"`
	TemplateKwargs map[string]string `yaml:"template_kwargs"`
	Jinja          bool              `yaml:"jinja"`
}

type RopeSpec struct {
	Scaling        string  `yaml:"scaling"`
	FreqBase       float64 `yaml:"freq_base"`
	FreqScale      float64 `yaml:"freq_scale"`
	YarnExtFactor  float64 `yaml:"yarn_ext_factor"`
	YarnAttnFactor float64 `yaml:"yarn_attn_factor"`
	YarnOrigCtx    int     `yaml:"yarn_orig_ctx"`
}

type ResourceSpec struct {
	VRAMLimit      string `yaml:"vram_limit"`
	VRAMBuffer     string `yaml:"vram_buffer"`
	MemoryLimit    string `yaml:"memory_limit"`
	CPULimit       string `yaml:"cpu_limit"`
	CPUPriority    string `yaml:"cpu_priority"`
	FallbackToCPU  bool   `yaml:"fallback_to_cpu"`
	RequestTimeout string `yaml:"request_timeout"`
	MaxConcurrent  int    `yaml:"max_concurrent"`
}

type LoggingSpec struct {
	Level  string `yaml:"level"`
	File   string `yaml:"file"`
	Colors string `yaml:"colors"`
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
}
