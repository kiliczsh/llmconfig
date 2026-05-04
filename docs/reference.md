# llmconfig — Documentation

**Local Large Model Config** — manage local inference with llama.cpp, stable-diffusion.cpp, and whisper.cpp.
Every command is also available via the shorter `llmc` alias.

## Table of Contents

- [Directory Layout](#directory-layout)
- [Getting Started](#getting-started)
- [Backends](#backends)
- [Commands](#commands)
- [Config File Reference](#config-file-reference)
- [Hardware Profiles](#hardware-profiles)
- [Environment Variables](#environment-variables)
- [OpenAI-Compatible API](#openai-compatible-api)

> **A note on paths.** This document writes directories as `~/.llmconfig/...`
> for brevity. That resolves to `$HOME/.llmconfig` on macOS and Linux and to
> `%USERPROFILE%\.llmconfig` on Windows. Set `LLMCONFIG_CONFIG_DIR` to
> override the base directory.

---

## Directory Layout

```
~/.llmconfig/
├── configs/          # YAML config files (<name>.yaml)
├── models/           # Downloaded model files (GGUF, whisper GGML, SD weights)
├── logs/             # Per-model log files (<name>.log)
├── bench/            # Saved benchmark results
├── bin/
│   ├── llama/        # llama.cpp binaries (managed by `llmconfig install llama`)
│   ├── sd/           # stable-diffusion.cpp binaries (`install sd`)
│   └── whisper/      # whisper.cpp binaries (`install whisper`)
└── state.json        # Running-model state
```

---

## Getting Started

### 1. Install a backend binary

```bash
llmconfig install llama       # text generation (llama.cpp)
llmconfig install sd          # image generation (stable-diffusion.cpp)
llmconfig install whisper     # speech recognition (whisper.cpp)
```

`install` auto-detects your hardware (CUDA, Metal, ROCm, CPU) and downloads
the matching release from GitHub. Binaries are placed under
`~/.llmconfig/bin/<backend>/`.

```bash
llmconfig llama --version    # verify the installed build
llmconfig llama --path       # show the binary path
```

The same `--version` / `--path` flags work for `llmconfig sd` and
`llmconfig whisper`.

### 2. Create a Config

**Option A — Interactive wizard:**

```bash
llmconfig init                            # full wizard (template vs. manual)
llmconfig init --template                 # template picker (bare flag)
llmconfig init --template=gemma           # use a specific template
llmconfig init my-llama --template=llama  # custom config name + template
llmconfig init --from bartowski/Meta-Llama-3.1-8B-Instruct-GGUF
```

> Pass template values with `--template=<name>` (with `=`). The bare
> `--template` form is reserved for the picker; a space-separated value
> would be parsed as the positional config name.

Built-in templates ship embedded in the binary. See
[templates.md](templates.md) for the full list with model details and
recommended quantizations. As of this writing the families are:

| Backend | Templates |
|---------|-----------|
| llama   | `gemma`, `llama`, `mistral`, `mistral-small`, `phi`, `phi4`, `phi4-reasoning`, `qwen`, `qwen36`, `qwen3-coder`, `qwen3-vl`, `granite4`, `deepseek`, `gpt-oss` |
| sd      | `sd`, `flux-schnell`, `flux-dev` |
| whisper | `whisper` |

`--template` with **no value** opens the interactive picker;
`--template <name>` skips the picker and uses that template directly. Run
`llmconfig init --template [TAB]` for shell completion of available names.

**Option B — Pull and auto-generate (llama only):**

```bash
llmconfig pull bartowski/Meta-Llama-3.1-8B-Instruct-GGUF --quant Q4_K_M
```

Downloads the model and creates a config in one step.

**Option C — Write manually:**

Create `<configs>/<name>.yaml` under your llmconfig directory (see
[Config File Reference](#config-file-reference)).

### 3. Start a Model

```bash
llmconfig up gemma-4-e2b
```

llmconfig will:
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
llmconfig down                # interactive selector if multiple models running
llmconfig down gemma-4-e2b   # stop by name
llmconfig down --all         # stop all running models
```

---

## Backends

llmconfig drives four inference backends. The `backend` field in a config
picks which one runs:

```yaml
backend: llama       # default — text generation via llama.cpp
# backend: ik_llama  # text generation via ik_llama.cpp (CPU/MoE-tuned fork)
# backend: sd        # image generation via stable-diffusion.cpp
# backend: whisper   # speech recognition via whisper.cpp
```

| Backend | Binary used | Install command | Managed bin dir |
|---------|-------------|-----------------|-----------------|
| `llama` (default) | `llama-server` (server) / `llama-cli` (interactive) | `llmconfig install llama` | `~/.llmconfig/bin/llama/` |
| `ik_llama` | `llama-server` / `llama-cli` (built from source) | `llmconfig install ik_llama` | `~/.llmconfig/bin/ik-llama/` |
| `sd` | `sd-cli` (server build) | `llmconfig install sd` | `~/.llmconfig/bin/sd/` |
| `whisper` | `whisper-server` / `whisper-cli` | `llmconfig install whisper` | `~/.llmconfig/bin/whisper/` |

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

### `ik_llama` — llama.cpp fork (ikawrakow)

A drop-in alternative to the `llama` backend that runs the
[ikawrakow/ik_llama.cpp](https://github.com/ikawrakow/ik_llama.cpp) fork.
The fork ships SOTA quants (IQ_K, trellis quants), MLA / FlashMLA for
DeepSeek, fused MoE kernels, and faster CPU and hybrid CPU/GPU paths.

**Build chain required.** Unlike the other backends, ik_llama.cpp publishes
no prebuilt release binaries. `llmconfig install ik_llama` clones the
source into `~/.llmconfig/cache/ik_llama.cpp/` and runs cmake. Prerequisites:

- `git`
- `cmake`
- A C++ compiler — gcc / clang on Linux/macOS, MSVC (`cl.exe` from a
  "Developer PowerShell for VS") on Windows
- CUDA toolkit if building with `--backend cuda`

Officially supported compute backends are CPU (AVX2+) and CUDA (Turing+).
ROCm, Vulkan, Metal, Intel GPU, and pre-AVX2 CPUs may compile but are
unsupported upstream.

```bash
llmconfig install ik_llama                       # auto-detect cpu vs cuda
llmconfig install ik_llama --backend cuda        # force CUDA
llmconfig install ik_llama --ref v0.1.2          # pin a tag/commit
llmconfig install ik_llama --jobs 4              # cap parallel jobs
llmconfig install ik_llama --verbose             # stream cmake output live
llmconfig install ik_llama --file build.zip      # bring-your-own-binary
```

The full build log is written to `~/.llmconfig/logs/install-ik-llama.log`.

```yaml
version: 1
name: ikqwen
backend: ik_llama

model:
  source: huggingface
  repo: bartowski/Qwen2.5-1.5B-Instruct-GGUF
  file: Qwen2.5-1.5B-Instruct-Q4_K_M.gguf

mode: server
server:
  host: 127.0.0.1
  port: 8080

hardware_profiles:
  cpu:
    n_gpu_layers: 0
  nvidia:
    n_gpu_layers: 99
    cuda: true

# Optional ik_llama-only flags. Omit this block entirely for stock behavior.
ik_llama:
  rtr: true            # -rtr  : run-time repack RAM-resident tensors (CPU quants)
  fmoe: true           # -fmoe : fused MoE matmul kernels
  mla: 2               # -mla N: multi-head latent attention level (DeepSeek)
  ser: "8,1"           # -ser N,thresh: smart expert reduction
  cuda_graphs: false   # → -cuda graphs=0 (workaround for split-mode graph corruption)
```

All the regular `llama` config blocks (`server`, `context`, `sampling`,
`hardware_profiles`, ...) work unchanged — ik_llama.cpp accepts the same
CLI flags as upstream llama.cpp.

> **Heads up on `-rtr`:** when running hybrid CPU/GPU inference for MoE
> models with experts left on CPU, do not enable `rtr` unless you know
> what you're doing. It forces matmuls on CPU-side tensors to stay on
> CPU even when GPU offload would be faster.

---

## Commands

> Commands marked with `*` show an interactive selector when called without a name and multiple models exist. If only one model is available it is selected automatically.

### `up <name>`

Start a model server. If the model is already running, prints its URL and exits successfully.

```bash
llmconfig up gemma-4-e2b
llmconfig up gemma-4-e2b --port 9000          # override port
llmconfig up gemma-4-e2b --profile cpu        # force hardware profile
llmconfig up gemma-4-e2b --dry-run            # print command, do not run
llmconfig up gemma-4-e2b --no-download        # fail if model not downloaded
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
llmconfig down                         # interactive selector
llmconfig down gemma-4-e2b            # stop by name
llmconfig down --all                  # stop all running models
llmconfig down gemma-4-e2b --timeout 30s
```

---

### `ps`

List running models.

```bash
llmconfig ps
llmconfig ps --all            # include stopped models
```

Output columns: `NAME`, `STATUS`, `PORT`, `PROFILE`, `UPTIME`, `PID`

---

### `logs [name]` `*`

Show model logs. Without a name, shows an interactive selector.

```bash
llmconfig logs                         # interactive selector
llmconfig logs gemma-4-e2b
llmconfig logs gemma-4-e2b -n 100
llmconfig logs gemma-4-e2b --follow
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
llmconfig stats
llmconfig stats gemma-4-e2b
llmconfig stats --watch
llmconfig stats --watch --interval 5s
```

---

### `status [name]` `*`

Detailed info for a single model. Without a name, shows an interactive selector.

```bash
llmconfig status                       # interactive selector
llmconfig status gemma-4-e2b
```

Shows: PID, port, profile, uptime, config path, log file path, status.

---

### `restart [name]` `*`

Stop and start a model, reloading config. Without a name, shows an interactive selector.

```bash
llmconfig restart                      # interactive selector
llmconfig restart gemma-4-e2b
llmconfig restart --all
```

---

### `pull <repo>`

Download a GGUF model from HuggingFace and create a config.

```bash
llmconfig pull bartowski/Meta-Llama-3.1-8B-Instruct-GGUF --quant Q4_K_M
llmconfig pull TheBloke/Mistral-7B-v0.1-GGUF --file mistral-7b-v0.1.Q4_K_M.gguf
llmconfig pull <repo> --token hf_xxx        # private repo
llmconfig pull <repo> --no-config           # download only, skip config creation
llmconfig pull <repo> --name my-custom-name
```

---

### `init [name]`

Interactive config wizard.

```bash
llmconfig init                                                      # full wizard
llmconfig init gemma-4-e2b                                          # set name up front
llmconfig init --template                                           # interactive template picker
llmconfig init --template=gemma                                     # use a specific template (note the =)
llmconfig init my-llama --template=llama                            # custom name + template
llmconfig init --from bartowski/google_gemma-4-E2B-it-GGUF
llmconfig init --from https://huggingface.co/.../model.gguf         # direct URL
llmconfig init --output ./gemma-4-e2b.yaml
```

The wizard first asks which backend to use (`llama`, `sd`, `whisper`), then
walks through backend-specific prompts. For llama it asks for: name,
HuggingFace repo, mode (server/interactive), port, and system prompt, then
lists available GGUF files from the repo for selection.

---

### `models`

List all known models (running, stopped, cached).

```bash
llmconfig models
llmconfig models --running
llmconfig models --cached
```

---

### `validate [name]` `*`

Validate a config without starting anything. Without a name, shows an interactive selector.

```bash
llmconfig validate                     # interactive selector
llmconfig validate gemma-4-e2b
llmconfig validate --file ./path/to/config.yaml
```

---

### `inspect [name]` `*`

Show the exact llama.cpp command that would be run. Without a name, shows an interactive selector.

```bash
llmconfig inspect                      # interactive selector
llmconfig inspect gemma-4-e2b
llmconfig inspect gemma-4-e2b --profile cpu    # inspect for a specific profile
```

Useful for debugging or running the command manually.

---

### `add <name>`

Register a local GGUF file as a named model.

```bash
llmconfig add gemma-4-e2b --path /path/to/model.gguf
llmconfig add gemma-4-e2b --path /path/to/model.gguf --copy   # copy to models dir
```

---

### `rm <name>`

Remove a model config (and optionally its downloaded file).

```bash
llmconfig rm gemma-4-e2b
llmconfig rm gemma-4-e2b --keep-file    # remove config, keep GGUF
llmconfig rm gemma-4-e2b --force        # skip confirmation prompt
```

---

### `config`

Manage config files.

```bash
llmconfig config list                 # list all configs
llmconfig config show gemma-4-e2b        # print resolved config (with defaults)
llmconfig config show gemma-4-e2b --raw  # print raw YAML without defaults
llmconfig config edit gemma-4-e2b        # open in $EDITOR
llmconfig config path gemma-4-e2b        # print file path
```

---

### `hardware`

Show detected hardware and the profile that would be selected.

```bash
llmconfig hardware
```

---

### `install <backend>`

Install a backend binary. Auto-detects your hardware and downloads the
matching release from GitHub.

```bash
llmconfig install llama               # text generation (llama.cpp)
llmconfig install ik_llama            # llama.cpp fork — built from source
llmconfig install sd                  # image generation (stable-diffusion.cpp)
llmconfig install whisper             # speech recognition (whisper.cpp)

llmconfig install llama --backend cuda  # force a specific build
llmconfig install llama --force         # reinstall even if already present
```

Each backend is installed into its own directory under
`~/.llmconfig/bin/<backend>/`, so they never collide. To update to the
latest release, run `install` again with `--force`. `ik_llama` works
differently — it builds from source rather than downloading a release;
see [`ik_llama` — llama.cpp fork](#ik_llama--llamacpp-fork-ikawrakow)
for prerequisites and flags.

Available `--backend` values depend on the backend and platform; common ones
are `cuda`, `metal`, `rocm`, and `cpu`.

---

### `llama` / `sd` / `whisper`

Show status for an installed backend binary.

```bash
llmconfig llama                       # show path + version
llmconfig llama --version             # print only the version line
llmconfig llama --path                # print the binary path

llmconfig sd --version
llmconfig whisper --path
```

If the binary is missing, these commands tell you to run
`llmconfig install <backend>`.

---

### `bench <name>`

Benchmark inference throughput for a model.

```bash
llmconfig bench gemma-4-e2b
llmconfig bench gemma-4-e2b --runs 3
llmconfig bench gemma-4-e2b --tokens 256
```

Results are written to `~/.llmconfig/bench/` for later comparison.

---

### `compat`

Show which configs fit on the detected hardware and estimate inference speed.

```bash
llmconfig compat
```

Analyses every config against detected RAM, VRAM, and bandwidth, so you can
see at a glance which models will actually run well on this machine.

---

### `version`

Print the llmconfig CLI version.

```bash
llmconfig version
llmconfig version --check     # also report whether a newer release exists
```

---

### `update`

Replace the running llmconfig binary with a newer release pulled from
GitHub. The download is verified against the published `checksums.txt`
before anything on disk is touched, and the previous binary is kept
beside the new one as `<binary>.old` so you can roll back manually if
needed.

```bash
llmconfig update                  # install the latest release
llmconfig update --check          # report whether an update is available
llmconfig update --version v1.2.0 # install (or downgrade to) a specific tag
llmconfig update --force          # reinstall even if already on target
```

If an `llmc` alias binary is found next to the main binary, it is
updated in the same operation.

The selfupdater only fetches assets from
`https://github.com/kiliczsh/llmconfig/releases` — every other host is
refused, and the SHA256 must match the one published in `checksums.txt`
or the install is aborted.

---

### `gateway`

Start a unified OpenAI-compatible HTTP gateway that routes requests to the correct running model based on the `"model"` parameter.

```bash
llmconfig gateway              # listen on default port 4000
llmconfig gateway --port 8000  # custom port
```

Flags:
| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port` | 4000 | Port to listen on |

Once running, you can target any model by name without knowing its individual port:

```bash
curl http://localhost:4000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gemma-4-e2b","messages":[{"role":"user","content":"Hello!"}]}'
```

`GET /v1/models` returns all currently running models. If the requested model is not running, the gateway returns `503 Service Unavailable`.

---

### `files`

Manage downloaded model files (`~/.llmconfig/models/`).

```bash
llmconfig files list             # list downloaded files with sizes (alias: ls)
llmconfig files clean            # remove files not referenced by any config
llmconfig files clean --all      # remove all downloaded files
llmconfig files path             # print models directory
```

---

### `state prune`

Maintenance command for the running-state file (`~/.llmconfig/state.json`).
If a model process was killed or crashed without llmconfig noticing, its
entry can linger as `running` even though the PID is dead. `state prune`
walks the state file, drops or marks any entry whose PID is no longer
alive, and reports what it changed.

```bash
llmconfig state prune
```

Use this if `llmconfig ps` shows a model as running that you can't actually
reach, or after a crash / forced shutdown.

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
    model_dir: ""           # defaults to ~/.llmconfig/models

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
# Override with: llmconfig up <name> --profile cpu

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
  file: ""          # defaults to ~/.llmconfig/logs/<name>.log
```

---

## Hardware Profiles

llmconfig auto-selects the hardware profile based on the detected system:

| Profile | Condition |
|---------|-----------|
| `apple_silicon` | macOS + ARM64 |
| `nvidia` | NVIDIA GPU detected via `nvidia-smi` |
| `amd` | AMD GPU detected via sysfs (Linux) or wmic (Windows) |
| `intel_gpu` | Intel GPU detected via sysfs |
| `cpu` | Fallback |

To see what was detected:

```bash
llmconfig hardware
```

To force a profile:

```bash
llmconfig up gemma-4-e2b --profile cpu
llmconfig inspect gemma-4-e2b --profile nvidia
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `LLMCONFIG_CONFIG_DIR` | Override the base directory (default: `~/.llmconfig`) |
| `HUGGINGFACE_TOKEN` | HuggingFace token for private repos (checked first) |
| `HF_TOKEN` | HuggingFace token, used when `HUGGINGFACE_TOKEN` is unset |
| `EDITOR` | Editor for `llmconfig config edit` |
| `VISUAL` | Fallback editor when `$EDITOR` is not set |

---

## OpenAI-Compatible API

When running in `server` mode, llmconfig exposes a standard OpenAI-compatible API:

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
