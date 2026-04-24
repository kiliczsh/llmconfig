# llmconfig

**Local Large Model Config** — manage local inference with llama.cpp, stable-diffusion.cpp, and whisper.cpp.

Define your model once in a YAML config file. llmconfig handles downloading, starting, stopping, and monitoring — across any hardware and across all three backends.

```
llmconfig up gemma-4-e2b        # or just: llmc up gemma-4-e2b
✓ gemma-4-e2b is ready at http://127.0.0.1:8080
```

> Ships with a shorter `llmc` alias — every command works with either binary name.

## Install

```bash
go install github.com/kiliczsh/llmconfig@latest
```

Or build from source:

```bash
git clone https://github.com/kiliczsh/llmconfig
cd llmconfig
go build -o llmconfig .
```

## Quick Start

```bash
# 1. Install the llama.cpp binary
llmconfig install llama

# 2. Create a config (interactive wizard)
llmconfig init --template gemma

# 3. Start the model
llmconfig up gemma-4-e2b

# 4. Send a request (OpenAI-compatible)
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gemma-4-e2b","messages":[{"role":"user","content":"Hello!"}]}'

# 5. Stop (no name needed if only one model is running)
llmconfig down
```

For image generation (stable-diffusion.cpp) or speech recognition
(whisper.cpp), swap the install step for `install sd` or `install whisper`
and choose a matching template in `init`. See [DOCS.md](DOCS.md#backends)
for details.

## Commands

Commands marked with `*` show an interactive selector when no name is given and multiple models exist.

| Command | Description |
|---------|-------------|
| `up <name>` | Start a model (reuses if already running) |
| `down [name]` `*` | Stop a running model |
| `ps` | List running models |
| `logs [name]` `*` | Show model logs |
| `stats [name]` `*` | Resource usage (supports `--watch`) |
| `status [name]` `*` | Detailed model info |
| `restart [name]` `*` | Stop and start |
| `pull <repo>` | Download from HuggingFace and create a config |
| `init [name]` | Create config interactively |
| `add <name>` | Add a config non-interactively via flags |
| `rm <name>` | Remove a config (and optionally its cache) |
| `models` | List all configured models |
| `inspect [name]` `*` | Show the backend command that would be executed |
| `validate [name]` `*` | Validate a config |
| `bench <name>` | Benchmark inference throughput |
| `compat` | Show which configs fit on detected hardware |
| `hardware` | Show detected hardware |
| `archive [name...]` `*` | Bundle config + cached model into a `.llmcpkg` file (interactive selector with no args) |
| `import <file.llmcpkg>` | Extract a `.llmcpkg` bundle back into configs + cache |
| `config list` | List all configs |
| `config show <name>` | Print a config with defaults applied |
| `config edit <name>` | Edit a config in `$EDITOR` |
| `config path <name>` | Print the resolved config file path |
| `cache list` \| `cache clean` \| `cache path` | Manage the downloaded-model cache |
| `install llama` \| `install sd` \| `install whisper` | Install a backend binary |
| `llama` \| `sd` \| `whisper` | Show backend status (supports `--version`, `--path`) |
| `version` | Show the llmconfig CLI version |

## Config File

Configs live in the llmconfig directory under your home folder
(`$HOME/.llmconfig/configs/<name>.yaml` on macOS/Linux,
`%USERPROFILE%\.llmconfig\configs\<name>.yaml` on Windows). Set
`LLMCONFIG_CONFIG_DIR` to override.

```yaml
version: 1
name: gemma-4-e2b
description: "Google Gemma 4 E2B instruct — 2B parameters, Q4_K_M quantization"

model:
  source: huggingface
  repo: bartowski/google_gemma-4-E2B-it-GGUF
  file: google_gemma-4-E2B-it-Q4_K_M.gguf
  download:
    resume: true
    connections: 4

mode: server  # or: interactive

server:
  host: 127.0.0.1
  port: 8080
  parallel: 1

hardware_profiles:
  nvidia:
    n_gpu_layers: 99
    cuda: true
    threads: 8
  apple_silicon:
    n_gpu_layers: 99
    metal: true
    threads: 8
  cpu:
    n_gpu_layers: 0
    threads: 8

context:
  n_ctx: 4096
  flash_attention: true
  mmap: true

sampling:
  temperature: 0.8
  top_p: 0.95
  top_k: 40
```

Hardware profile is selected automatically at runtime. See [DOCS.md](DOCS.md) for the full config reference.

## Requirements

- Go 1.26+
- A backend binary — `llmconfig install llama` (or `install sd` /
  `install whisper`) downloads and installs the right build for your
  hardware.

## License

MIT
