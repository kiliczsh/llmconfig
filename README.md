# llmconfig

**Local Large Model Config** — manage local inference with `llama.cpp`,
`stable-diffusion.cpp`, and `whisper.cpp` from a single YAML file and a
single CLI.

```
llmconfig up gemma                    # or just: llmc up gemma
✓ gemma is ready at http://127.0.0.1:8080
```

> Ships with a shorter `llmc` alias — every command works with either binary name.

---

## Why llmconfig

- **One YAML, three backends.** Define a model once; llmconfig handles
  downloading, starting, stopping, restarting, and monitoring across
  text, image, and speech backends.
- **Hardware-aware.** Profiles for NVIDIA, Apple Silicon, AMD, Intel GPU,
  and CPU are auto-selected at runtime. Override with `--profile`.
- **OpenAI-compatible.** Models run as drop-in replacements for the OpenAI
  API. The optional `gateway` command exposes every running model on a
  single port.
- **No build chain.** Backend binaries are downloaded and managed for you;
  `llmconfig install llama` (or `sd` / `whisper`) is a one-shot.

---

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

Requires **Go 1.26+** (see `go.mod`).

---

## Quick Start

```bash
# 1. Install the llama.cpp binary (CUDA / Metal / CPU build auto-detected)
llmconfig install llama

# 2. Create a config from a built-in template (note the `=` — see Templates below)
llmconfig init --template=gemma

# 3. Start the model
llmconfig up gemma

# 4. Send a request — OpenAI-compatible
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gemma","messages":[{"role":"user","content":"Hello!"}]}'

# 5. Stop (no name needed if only one model is running)
llmconfig down
```

For image generation or speech recognition, swap step 1 for `install sd`
or `install whisper` and pick a matching template in step 2.
See [Backends](#backends) and [DOCS.md](DOCS.md#backends) for details.

---

## Backends

| Backend | Engine | Install | Default port |
|---------|--------|---------|--------------|
| `llama` | [llama.cpp](https://github.com/ggerganov/llama.cpp) — text | `llmconfig install llama` | 8080 |
| `sd` | [stable-diffusion.cpp](https://github.com/leejet/stable-diffusion.cpp) — image | `llmconfig install sd` | 8091 |
| `whisper` | [whisper.cpp](https://github.com/ggerganov/whisper.cpp) — speech-to-text | `llmconfig install whisper` | 8082 |

Each backend reads its own block in the YAML config (`server`, `sd`,
`whisper`) plus the shared fields (`model`, `hardware_profiles`,
`context`, ...). See [DOCS.md → Config File Reference](DOCS.md#config-file-reference)
for the full schema.

---

## Built-in templates

```bash
llmconfig init --template               # interactive picker (no value)
llmconfig init --template=gemma         # skip picker, use this template
llmconfig init my-name --template=llama # custom config name + template
```

> Use `--template=<name>` (with `=`) when passing a value. The bare
> `--template` form opens the picker by design, so a space-separated
> value would be parsed as the positional config name instead.

Sizes below are the recommended quantization for ~16 GB VRAM; each
template ships with commented alternatives for other VRAM budgets.

### Text — `llama` backend

| Template | Model | Recommended size |
|----------|-------|------------------|
| `gemma` | Google Gemma 4 E4B Instruct | ~4.4 GB |
| `llama` | Meta Llama 3.1 8B Instruct | ~8.7 GB |
| `mistral` | Mistral 7B Instruct v0.3 | ~2.4 GB |
| `mistral-small` | Mistral Small 3.2 24B Instruct | ~11.4 GB |
| `phi` | Microsoft Phi-4 Mini Instruct | ~2.3 GB |
| `phi4` | Microsoft Phi-4 (14B) | ~8.4 GB |
| `phi4-reasoning` | Microsoft Phi-4 Reasoning Plus | ~11.7 GB |
| `qwen` | Qwen 2.5 1.5B Instruct | ~1.0 GB |
| `qwen36` | Qwen 3.6 27B | ~12.8 GB |
| `qwen3-coder` | Qwen 3 Coder 30B A3B (MoE) | ~12 GB |
| `qwen3-vl` | Qwen 3 VL 8B Instruct (vision-language) | ~6.7 GB |
| `granite4` | IBM Granite 4.0 H Tiny | ~5.9 GB |
| `deepseek` | DeepSeek R1 0528 Qwen3 8B (reasoning) | ~8.7 GB |
| `gpt-oss` | GPT-OSS 20B | ~12.3 GB |

### Image — `sd` backend

| Template | Model | Notes |
|----------|-------|-------|
| `sd` | Stable Diffusion 1.5 (RunwayML) | Classic checkpoint, 512×512 |
| `flux-schnell` | Black Forest Labs FLUX.1 Schnell | Distilled, 4 steps, 1024×1024 |
| `flux-dev` | Black Forest Labs FLUX.1 Dev | Higher quality, 20 steps, 1024×1024 |

### Speech — `whisper` backend

| Template | Model | Notes |
|----------|-------|-------|
| `whisper` | OpenAI Whisper (ggml) | Defaults to `base`; pick `medium`, `large-v3-turbo`, etc. |

---

## Commands

Commands marked with `*` show an interactive selector when no name is given and multiple models exist.

| Command | Description |
|---------|-------------|
| `up <name>` | Start a model (reuses if already running) |
| `down [name]` `*` | Stop a running model |
| `restart [name]` `*` | Stop and start |
| `ps` | List running models |
| `logs [name]` `*` | Show model logs |
| `stats [name]` | Resource usage (supports `--watch`, `--interval`) |
| `status [name]` `*` | Detailed model info |
| `pull <repo>` | Download from HuggingFace and create a config |
| `init [name]` | Create config interactively (`--template` for picker, `--template <name>` to skip) |
| `add <name>` | Add a config non-interactively via flags |
| `rm <name>` | Remove a config (and optionally its cache) |
| `models` | List all configured models |
| `inspect [name]` `*` | Show the backend command that would be executed |
| `validate [name]` `*` | Validate a config |
| `bench <name>` | Benchmark inference throughput |
| `compat` | Show which configs fit on detected hardware |
| `hardware` | Show detected hardware |
| `archive [name...]` `*` | Bundle config + cached model into a `.llmcpkg` file |
| `import <file.llmcpkg>` | Extract a `.llmcpkg` bundle back into configs + cache |
| `config list` \| `config show <name>` \| `config edit <name>` \| `config path <name>` | Manage config files |
| `files list` \| `files clean` \| `files path` | Manage the downloaded-model cache |
| `gateway` | Start a unified OpenAI-compatible gateway for all running models |
| `state prune` | Mark dead-PID running entries as stopped |
| `install llama` \| `install sd` \| `install whisper` | Install a backend binary |
| `llama` \| `sd` \| `whisper` | Show backend status (supports `--version`, `--path`) |
| `version` | Show the llmconfig CLI version |

Every command has `--help`; full reference with flags and examples lives in [DOCS.md](DOCS.md#commands).

---

## Config example

Configs live under your home directory:

- macOS / Linux: `$HOME/.llmconfig/configs/<name>.yaml`
- Windows: `%USERPROFILE%\.llmconfig\configs\<name>.yaml`
- Override the base directory with `LLMCONFIG_CONFIG_DIR`.

```yaml
version: 1
name: gemma
description: "Google Gemma 4 E4B Instruct"

backend: llama
mode: server                  # or: interactive

model:
  source: huggingface
  repo: unsloth/gemma-4-E4B-it-GGUF
  file: gemma-4-E4B-it-UD-Q8_K_XL.gguf
  download:
    resume: true

server:
  host: 127.0.0.1
  port: 8080

hardware_profiles:
  nvidia:        { n_gpu_layers: 99, cuda: true }
  apple_silicon: { n_gpu_layers: 99, metal: true }
  cpu:           { n_gpu_layers: 0 }

context:
  n_ctx: 4096
  flash_attention: true

sampling:
  temperature: 0.8
  top_p: 0.95
  top_k: 40
```

The matching hardware profile is selected automatically at runtime
(override with `--profile`). See [DOCS.md → Config File Reference](DOCS.md#config-file-reference)
for every supported field.

---

## Documentation

- [DOCS.md](DOCS.md) — full reference: every command, every config field, every environment variable.
- [CHANGELOG.md](CHANGELOG.md) — what's changed between releases.
- [CONTRIBUTING.md](CONTRIBUTING.md) — set up the project, add a template, send a PR.

---

## Contributing

Contributions are welcome — bug reports, new templates, and PRs alike.
Start with [CONTRIBUTING.md](CONTRIBUTING.md) for project layout, build
steps, and the template authoring guide. Use the
[issue templates](.github/ISSUE_TEMPLATE/) when filing bugs or feature
requests.

---

## License

[MIT](LICENSE)
