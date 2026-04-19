# llamaconfig

Manage local LLM inference with llama.cpp.

Define your model once in a YAML config file. llamaconfig handles downloading, starting, stopping, and monitoring — across any hardware.

```
llamaconfig up gemma-4-e2b
✓ gemma-4-e2b is ready at http://127.0.0.1:8080
```

## Install

```bash
go install github.com/kiliczsh/llamaconfig@latest
```

Or build from source:

```bash
git clone https://github.com/kiliczsh/llamaconfig
cd llamaconfig
go build -o llamaconfig .
```

## Quick Start

```bash
# 1. Install llama.cpp binary
llamaconfig llama --install

# 2. Create a config (interactive wizard)
llamaconfig init --template gemma

# 3. Start the model
llamaconfig up gemma-4-e2b

# 4. Send a request (OpenAI-compatible)
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gemma-4-e2b","messages":[{"role":"user","content":"Hello!"}]}'

# 5. Stop
llamaconfig down gemma-4-e2b
```

## Commands

| Command | Description |
|---------|-------------|
| `up <name>` | Start a model |
| `down [name]` | Stop a model |
| `ps` | List running models |
| `logs <name>` | Show model logs |
| `stats` | Resource usage |
| `status <name>` | Detailed model info |
| `restart <name>` | Stop and start |
| `pull <repo>` | Download from HuggingFace |
| `init` | Create config interactively |
| `models` | List all models |
| `inspect <name>` | Show llama.cpp command |
| `validate <name>` | Validate config |
| `hardware` | Show detected hardware |
| `config list` | List all configs |
| `config edit <name>` | Edit config in $EDITOR |
| `cache ls` | List cached files |
| `llama --install` | Install llama.cpp binary |

## Config File

Configs live in `~/.llamaconfig/configs/<name>.yaml`.

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
  temperature: 0.7
  top_p: 0.95
  top_k: 40

context:
  n_ctx: 4096
  flash_attention: true
```

Hardware profile is selected automatically at runtime. See [DOCS.md](DOCS.md) for the full config reference.

## Requirements

- Go 1.21+
- llama.cpp binary (`llamaconfig llama --install` handles this)

## License

MIT
