# llamaconfig — Documentation

## Table of Contents

- [Directory Layout](#directory-layout)
- [Getting Started](#getting-started)
- [Backends](#backends)
- [Commands](#commands)
- [Config File Reference](#config-file-reference)
- [Hardware Profiles](#hardware-profiles)
- [Environment Variables](#environment-variables)
- [OpenAI-Compatible API](#openai-compatible-api)

> **A note on paths.** This document writes directories as `~/.llamaconfig/...`
> for brevity. That resolves to `$HOME/.llamaconfig` on macOS and Linux and to
> `%USERPROFILE%\.llamaconfig` on Windows. Set `LLAMACONFIG_CONFIG_DIR` to
> override the base directory.

---

## Directory Layout

```
~/.llamaconfig/
├── configs/          # YAML config files (<name>.yaml)
├── cache/            # Downloaded model files (GGUF, whisper GGML, SD weights)
├── logs/             # Per-model log files (<name>.log)
├── bench/            # Saved benchmark results
├── bin/
│   ├── llama/        # llama.cpp binaries (managed by `llamaconfig install llama`)
│   ├── sd/           # stable-diffusion.cpp binaries (`install sd`)
│   └── whisper/      # whisper.cpp binaries (`install whisper`)
└── state.json        # Running-model state
```

---

## Getting Started

### 1. Install a backend binary

```bash
llamaconfig install llama       # text generation (llama.cpp)
llamaconfig install sd          # image generation (stable-diffusion.cpp)
llamaconfig install whisper     # speech recognition (whisper.cpp)
```

`install` auto-detects your hardware (CUDA, Metal, ROCm, CPU) and downloads
the matching release from GitHub. Binaries are placed under
`~/.llamaconfig/bin/<backend>/`.

```bash
llamaconfig llama --version    # verify the installed build
llamaconfig llama --path       # show the binary path
```

The same `--version` / `--path` flags work for `llamaconfig sd` and
`llamaconfig whisper`.

### 2. Create a Config

**Option A — Interactive wizard:**

```bash
llamaconfig init
llamaconfig init --template llama3    # pre-fill with a known model
llamaconfig init --from bartowski/Meta-Llama-3.1-8B-Instruct-GGUF
```

Built-in templates:

| Backend | Templates |
|---------|-----------|
| llama   | `codellama`, `mistral`, `llama3`, `deepseek`, `phi4`, `gemma` |
| sd      | `sd15`, `flux-schnell`, `flux-dev` |
| whisper | `whisper-base`, `whisper-turbo` |

The wizard asks which backend to use; pass `--template` to skip the picker.

**Option B — Pull and auto-generate (llama only):**

```bash
llamaconfig pull bartowski/Meta-Llama-3.1-8B-Instruct-GGUF --quant Q4_K_M
```

Downloads the model and creates a config in one step.

**Option C — Write manually:**

Create `<configs>/<name>.yaml` under your llamaconfig directory (see
[Config File Reference](#config-file-reference)).

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

## Backends

llamaconfig drives three inference backends. The `backend` field in a config
picks which one runs:

```yaml
backend: llama      # default — text generation via llama.cpp
# backend: sd       # image generation via stable-diffusion.cpp
# backend: whisper  # speech recognition via whisper.cpp
```

| Backend | Binary used | Install command | Managed bin dir |
|---------|-------------|-----------------|-----------------|
| `llama` (default) | `llama-server` (server) / `llama-cli` (interactive) | `llamaconfig install llama` | `~/.llamaconfig/bin/llama/` |
| `sd` | `sd-cli` (server build) | `llamaconfig install sd` | `~/.llamaconfig/bin/sd/` |
| `whisper` | `whisper-server` / `whisper-cli` | `llamaconfig install whisper` | `~/.llamaconfig/bin/whisper/` |

Each backend reads a backend-specific config block in addition to the shared
fields (`model`, `server`, `hardware_profiles`, etc.).

### `sd` — Stable Diffusion

```yaml
version: 1
name: flux-schnell
backend: sd

model:
  source: huggingface
  repo: city96/FLUX.1-schnell-gguf
  file: flux1-schnell-Q4_K_S.gguf

mode: server
server:
  host: 127.0.0.1
  port: 8090

sd:
  width: 512
  height: 512
  steps: 20
  cfg_scale: 7.0
  sampling_method: euler_a   # euler_a | euler | dpm++2m | lcm | ...
  seed: -1                    # -1 = random
```

### `whisper` — Speech Recognition

```yaml
version: 1
name: whisper-turbo
backend: whisper

model:
  source: huggingface
  repo: ggerganov/whisper.cpp
  file: ggml-large-v3-turbo.bin

mode: server
server:
  host: 127.0.0.1
  port: 8082

whisper:
  language: auto        # auto | en | tr | ...
  task: transcribe      # transcribe | translate
  beam_size: 5
  best_of: 5
  vad: true
  vad_threshold: 0.5
  word_timestamps: false
  processors: 1
```

See [Config File Reference](#config-file-reference) for the shared fields
(`model`, `server`, `hardware_profiles`, `context`, ...).

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
```

---

### `status [name]` `*`

Detailed info for a single model. Without a name, shows an interactive selector.

```bash
llamaconfig status                       # interactive selector
llamaconfig status gemma-4-e2b
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
llamaconfig init --from https://huggingface.co/.../model.gguf   # direct URL
llamaconfig init --output ./gemma-4-e2b.yaml
```

The wizard first asks which backend to use (`llama`, `sd`, `whisper`), then
walks through backend-specific prompts. For llama it asks for: name,
HuggingFace repo, mode (server/interactive), port, and system prompt, then
lists available GGUF files from the repo for selection.

---

### `models`

List all known models (running, stopped, cached).

```bash
llamaconfig models
llamaconfig models --running
llamaconfig models --cached
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
```

---

### `install <backend>`

Install a backend binary. Auto-detects your hardware and downloads the
matching release from GitHub.

```bash
llamaconfig install llama               # text generation (llama.cpp)
llamaconfig install sd                  # image generation (stable-diffusion.cpp)
llamaconfig install whisper             # speech recognition (whisper.cpp)

llamaconfig install llama --backend cuda  # force a specific build
llamaconfig install llama --force         # reinstall even if already present
```

Each backend is installed into its own directory under
`~/.llamaconfig/bin/<backend>/`, so they never collide. To update to the
latest release, run `install` again with `--force`.

Available `--backend` values depend on the backend and platform; common ones
are `cuda`, `metal`, `rocm`, and `cpu`.

---

### `llama` / `sd` / `whisper`

Show status for an installed backend binary.

```bash
llamaconfig llama                       # show path + version
llamaconfig llama --version             # print only the version line
llamaconfig llama --path                # print the binary path

llamaconfig sd --version
llamaconfig whisper --path
```

If the binary is missing, these commands tell you to run
`llamaconfig install <backend>`.

---

### `bench <name>`

Benchmark inference throughput for a model.

```bash
llamaconfig bench gemma-4-e2b
llamaconfig bench gemma-4-e2b --runs 3
llamaconfig bench gemma-4-e2b --tokens 256
```

Results are written to `~/.llamaconfig/bench/` for later comparison.

---

### `compat`

Show which configs fit on the detected hardware and estimate inference speed.

```bash
llamaconfig compat
```

Analyses every config against detected RAM, VRAM, and bandwidth, so you can
see at a glance which models will actually run well on this machine.

---

### `version`

Print the llamaconfig CLI version.

```bash
llamaconfig version
```

---

### `cache`

Manage the model file cache (`~/.llamaconfig/cache/`).

```bash
llamaconfig cache list             # list cached files with sizes (alias: ls)
llamaconfig cache clean            # remove files not referenced by any config
llamaconfig cache clean --all      # remove all cached files
llamaconfig cache path             # print cache directory
```

---

## Config File Reference

The example below shows the shared fields used by every backend (`model`,
`server`, `hardware_profiles`, `context`, `sampling`, ...) plus llama-only
fields. For `sd` and `whisper` specifics, see [Backends](#backends).

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
    resume: true            # default: true — resume interrupted downloads
    connections: 4          # parallel download connections (default: 4)
    verify_checksum: true   # default: true — set to false to skip checksum verification
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
    metrics: false          # default: false — enables --metrics in llama-server
    slots: true             # default: true — set to false to pass --no-slots
    embeddings: false       # default: false — enables --embedding in llama-server
    # health, completions, chat — reserved for future use; currently ignored

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
  temperature: 0.8        # default: 0.8
  top_k: 40               # default: 40
  top_p: 0.95             # default: 0.95
  min_p: 0.05             # default: 0.05
  repeat_penalty: 1.0     # default: 1.0
  repeat_last_n: 64       # default: 64

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
| `LLAMACONFIG_CONFIG_DIR` | Override the base directory (default: `~/.llamaconfig`) |
| `HUGGINGFACE_TOKEN` | HuggingFace token for private repos (checked first) |
| `HF_TOKEN` | HuggingFace token, used when `HUGGINGFACE_TOKEN` is unset |
| `EDITOR` | Editor for `llamaconfig config edit` |
| `VISUAL` | Fallback editor when `$EDITOR` is not set |

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
