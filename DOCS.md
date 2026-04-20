# llamaconfig — Documentation

## Table of Contents

- [Directory Layout](#directory-layout)
- [Getting Started](#getting-started)
- [Commands](#commands)
- [Config File Reference](#config-file-reference)
- [Hardware Profiles](#hardware-profiles)
- [Environment Variables](#environment-variables)
- [OpenAI-Compatible API](#openai-compatible-api)

---

## Directory Layout

```
~/.llamaconfig/
├── configs/        # YAML config files (<name>.yaml)
├── cache/          # Downloaded GGUF model files
├── logs/           # Per-model log files (<name>.log)
├── bin/            # llama.cpp binaries (managed by llamaconfig llama --install)
└── state.json      # Running model state
```

---

## Getting Started

### 1. Install llama.cpp

```bash
llamaconfig llama --install
```

Auto-detects your hardware (CUDA, Metal, CPU) and downloads the matching llama.cpp release from GitHub. Binaries are placed in `~/.llamaconfig/bin/`.

```bash
llamaconfig llama --version    # verify
llamaconfig llama --path       # show binary path
```

### 2. Create a Config

**Option A — Interactive wizard:**

```bash
llamaconfig init
llamaconfig init --template llama3    # pre-fill with a known model
llamaconfig init --from bartowski/Meta-Llama-3.1-8B-Instruct-GGUF
```

Built-in templates: `codellama`, `mistral`, `llama3`, `deepseek`, `phi4`, `gemma`

**Option B — Pull and auto-generate:**

```bash
llamaconfig pull bartowski/Meta-Llama-3.1-8B-Instruct-GGUF --quant Q4_K_M
```

Downloads the model and creates a config in one step.

**Option C — Write manually:**

Create `~/.llamaconfig/configs/<name>.yaml` (see [Config File Reference](#config-file-reference)).

### 3. Start a Model

```bash
llamaconfig up gemma-4-e2b
```

llamaconfig will:
1. Load and validate the config
2. Detect hardware and select the matching profile
3. Download the model file if not cached
4. Start `llama-server` in the background
5. Poll `/health` until the server is ready

### 4. Use the API

The server exposes an OpenAI-compatible REST API:

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemma-4-e2b",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### 5. Stop

```bash
llamaconfig down                # interactive selector if multiple models running
llamaconfig down gemma-4-e2b   # stop by name
llamaconfig down --all         # stop all running models
```

---

## Commands

> Commands marked with `*` show an interactive selector when called without a name and multiple models exist. If only one model is available it is selected automatically.

### `up <name>`

Start a model server. If the model is already running, prints its URL and exits successfully.

```bash
llamaconfig up gemma-4-e2b
llamaconfig up gemma-4-e2b --port 9000          # override port
llamaconfig up gemma-4-e2b --profile cpu        # force hardware profile
llamaconfig up gemma-4-e2b --dry-run            # print command, do not run
llamaconfig up gemma-4-e2b --no-download        # fail if model not cached
```

Flags:
| Flag | Default | Description |
|------|---------|-------------|
| `--port` | config value | Override server port |
| `--profile` | auto | Force profile: `nvidia` \| `apple_silicon` \| `amd` \| `cpu` |
| `--dry-run` | false | Print llama.cpp command without starting |
| `--no-download` | false | Fail instead of downloading |
| `-d, --detach` | true | Run in background |

---

### `down [name]` `*`

Stop a running model. Without a name, shows an interactive selector.

```bash
llamaconfig down                         # interactive selector
llamaconfig down gemma-4-e2b            # stop by name
llamaconfig down --all                  # stop all running models
llamaconfig down gemma-4-e2b --timeout 30s
```

---

### `ps`

List running models.

```bash
llamaconfig ps
llamaconfig ps --all            # include stopped models
llamaconfig ps --format json
```

Output columns: `NAME`, `STATUS`, `PORT`, `PROFILE`, `UPTIME`, `PID`

---

### `logs [name]` `*`

Show model logs. Without a name, shows an interactive selector.

```bash
llamaconfig logs                         # interactive selector
llamaconfig logs gemma-4-e2b
llamaconfig logs gemma-4-e2b -n 100
llamaconfig logs gemma-4-e2b --follow
```

Flags:
| Flag | Default | Description |
|------|---------|-------------|
| `-n, --lines` | 50 | Number of lines to show |
| `-f, --follow` | false | Stream in real time |

---

### `stats [name]`

Show CPU and memory usage of running models.

```bash
llamaconfig stats
llamaconfig stats gemma-4-e2b
llamaconfig stats --watch
llamaconfig stats --watch --interval 5s
llamaconfig stats --format json
```

---

### `status [name]` `*`

Detailed info for a single model. Without a name, shows an interactive selector.

```bash
llamaconfig status                       # interactive selector
llamaconfig status gemma-4-e2b
llamaconfig status gemma-4-e2b --json
```

Shows: PID, port, profile, uptime, config path, log file path, status.

---

### `restart [name]` `*`

Stop and start a model, reloading config. Without a name, shows an interactive selector.

```bash
llamaconfig restart                      # interactive selector
llamaconfig restart gemma-4-e2b
llamaconfig restart --all
```

---

### `pull <repo>`

Download a GGUF model from HuggingFace and create a config.

```bash
llamaconfig pull bartowski/Meta-Llama-3.1-8B-Instruct-GGUF --quant Q4_K_M
llamaconfig pull TheBloke/Mistral-7B-v0.1-GGUF --file mistral-7b-v0.1.Q4_K_M.gguf
llamaconfig pull <repo> --token hf_xxx        # private repo
llamaconfig pull <repo> --no-config           # download only, skip config creation
llamaconfig pull <repo> --name my-custom-name
```

---

### `init [name]`

Interactive config wizard.

```bash
llamaconfig init
llamaconfig init gemma-4-e2b
llamaconfig init --template llama3
llamaconfig init --from bartowski/google_gemma-4-E2B-it-GGUF
llamaconfig init --output ./gemma-4-e2b.yaml
```

The wizard prompts for: name, HuggingFace repo, mode (server/interactive), port, system prompt. Then lists available GGUF files from the repo for selection.

---

### `models`

List all known models (running, stopped, cached).

```bash
llamaconfig models
llamaconfig models --running
llamaconfig models --cached
llamaconfig models --format json
```

---

### `validate [name]` `*`

Validate a config without starting anything. Without a name, shows an interactive selector.

```bash
llamaconfig validate                     # interactive selector
llamaconfig validate gemma-4-e2b
llamaconfig validate --file ./path/to/config.yaml
```

---

### `inspect [name]` `*`

Show the exact llama.cpp command that would be run. Without a name, shows an interactive selector.

```bash
llamaconfig inspect                      # interactive selector
llamaconfig inspect gemma-4-e2b
llamaconfig inspect gemma-4-e2b --profile cpu    # inspect for a specific profile
```

Useful for debugging or running the command manually.

---

### `add <name>`

Register a local GGUF file as a named model.

```bash
llamaconfig add gemma-4-e2b --path /path/to/model.gguf
llamaconfig add gemma-4-e2b --path /path/to/model.gguf --copy   # copy to cache dir
```

---

### `rm <name>`

Remove a model config (and optionally its cached file).

```bash
llamaconfig rm gemma-4-e2b
llamaconfig rm gemma-4-e2b --keep-file    # remove config, keep GGUF
llamaconfig rm gemma-4-e2b --force        # skip confirmation prompt
```

---

### `config`

Manage config files.

```bash
llamaconfig config list                 # list all configs
llamaconfig config show gemma-4-e2b        # print resolved config (with defaults)
llamaconfig config show gemma-4-e2b --raw  # print raw YAML without defaults
llamaconfig config edit gemma-4-e2b        # open in $EDITOR
llamaconfig config path gemma-4-e2b        # print file path
```

---

### `hardware`

Show detected hardware and the profile that would be selected.

```bash
llamaconfig hardware
llamaconfig hardware --json
```

---

### `llama`

Manage the llama.cpp binary.

```bash
llamaconfig llama --install             # install latest release
llamaconfig llama --update              # update to latest
llamaconfig llama --version             # print version
llamaconfig llama --path                # print binary path
llamaconfig llama --install --backend cuda    # force backend
```

Backends: `cuda`, `metal`, `cpu`

---

### `cache`

Manage the model file cache (`~/.llamaconfig/cache/`).

```bash
llamaconfig cache ls               # list cached files with sizes
llamaconfig cache clean            # remove files not referenced by any config
llamaconfig cache clean --all      # remove all cached files
llamaconfig cache path             # print cache directory
```

---

## Config File Reference

Full example with all supported fields:

```yaml
version: 1
name: gemma-4-e2b
description: "Optional description"
tags: [llama, instruct, 8b]

# ── Model ──────────────────────────────────────────────────────────────────

model:
  source: huggingface       # huggingface | url | local
  repo: bartowski/Meta-Llama-3.1-8B-Instruct-GGUF
  file: Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf
  # path: /absolute/path/to/model.gguf   # for source: local
  # url: https://example.com/model.gguf  # for source: url
  checksum: sha256:abc123...             # optional, verified on download

  download:
    resume: true
    connections: 4          # parallel download connections
    verify_checksum: false
    cache_dir: ""           # defaults to ~/.llamaconfig/cache

  # Speculative decoding (optional)
  draft:
    source: huggingface
    repo: bartowski/some-draft-GGUF
    file: draft-model.gguf
    draft_n: 5

# ── Mode ───────────────────────────────────────────────────────────────────

mode: server          # server | interactive
# server    → starts llama-server, OpenAI-compatible HTTP API
# interactive → starts llama-cli, terminal chat session

# ── Server ─────────────────────────────────────────────────────────────────

server:
  host: 127.0.0.1
  port: 8080
  api_key: ""               # optional bearer token
  parallel: 1               # concurrent request slots
  cors_origins: []

  endpoints:
    metrics: false
    slots: false
    health: true
    completions: true
    chat: true
    embeddings: false

# ── Hardware Profiles ──────────────────────────────────────────────────────
# The matching profile is selected automatically at runtime.
# Override with: llamaconfig up <name> --profile cpu

hardware_profiles:
  apple_silicon:
    n_gpu_layers: 99
    metal: true
    threads: 8

  nvidia:
    n_gpu_layers: 99
    cuda: true
    threads: 8

  amd:
    n_gpu_layers: 99
    rocm: true
    threads: 8

  intel_gpu:
    n_gpu_layers: 99
    sycl: true
    threads: 8

  cpu:
    n_gpu_layers: 0
    threads: 8

# ── Context ────────────────────────────────────────────────────────────────

context:
  n_ctx: 4096           # context window size
  n_batch: 512          # prompt processing batch size
  n_ubatch: 512         # physical batch size
  n_keep: 0
  cache_type_k: f16
  cache_type_v: f16
  mmap: true
  mlock: false
  flash_attention: true

# ── Sampling ───────────────────────────────────────────────────────────────

sampling:
  temperature: 0.7
  top_k: 40
  top_p: 0.95
  min_p: 0.05
  repeat_penalty: 1.1
  repeat_last_n: 64

# ── Chat ───────────────────────────────────────────────────────────────────

chat:
  system_prompt: |
    You are a helpful assistant.
  template: ""      # override chat template (leave empty to use model default)
  jinja: false

# ── RoPE ───────────────────────────────────────────────────────────────────

rope:
  scaling: ""
  freq_base: 0
  freq_scale: 0

# ── Logging ────────────────────────────────────────────────────────────────

logging:
  level: info
  file: ""          # defaults to ~/.llamaconfig/logs/<name>.log
```

---

## Hardware Profiles

llamaconfig auto-selects the hardware profile based on the detected system:

| Profile | Condition |
|---------|-----------|
| `apple_silicon` | macOS + ARM64 |
| `nvidia` | NVIDIA GPU detected via `nvidia-smi` |
| `amd` | AMD GPU detected via sysfs (Linux) or wmic (Windows) |
| `intel_gpu` | Intel GPU detected via sysfs |
| `cpu` | Fallback |

To see what was detected:

```bash
llamaconfig hardware
```

To force a profile:

```bash
llamaconfig up gemma-4-e2b --profile cpu
llamaconfig inspect gemma-4-e2b --profile nvidia
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `LLAMACONFIG_CONFIG_DIR` | Override config directory (default: `~/.llamaconfig`) |
| `HF_TOKEN` | HuggingFace token for private repos |
| `EDITOR` | Editor for `llamaconfig config edit` |
| `VISUAL` | Fallback editor if `$EDITOR` is not set |

---

## OpenAI-Compatible API

When running in `server` mode, llamaconfig exposes a standard OpenAI-compatible API:

```
POST /v1/chat/completions
POST /v1/completions
POST /v1/embeddings
GET  /v1/models
GET  /health
```

This means it works as a drop-in with any OpenAI SDK:

**Python (openai SDK):**
```python
from openai import OpenAI

client = OpenAI(base_url="http://127.0.0.1:8080/v1", api_key="none")

response = client.chat.completions.create(
    model="gemma-4-e2b",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)
```

**Node.js:**
```javascript
import OpenAI from "openai";

const client = new OpenAI({ baseURL: "http://127.0.0.1:8080/v1", apiKey: "none" });

const response = await client.chat.completions.create({
  model: "gemma-4-e2b",
  messages: [{ role: "user", content: "Hello!" }],
});
console.log(response.choices[0].message.content);
```

**curl:**
```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemma-4-e2b",
    "messages": [{"role": "user", "content": "Hello!"}],
    "temperature": 0.7,
    "max_tokens": 512
  }'
```
