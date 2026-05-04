# llmconfig

**Local Large Model Config** — manage local inference with `llama.cpp`,
`stable-diffusion.cpp`, and `whisper.cpp` from a single YAML file and a
single CLI.

```
llmconfig up gemma                    # or just: llmc up gemma
✓ gemma is ready at http://127.0.0.1:8080
```

> Ships with a shorter `llmc` alias — every command works with either binary name.

## Why llmconfig

- **One YAML, three backends.** Define a model once; llmconfig handles
  downloading, starting, stopping, restarting, and monitoring.
- **Hardware-aware.** Profiles for NVIDIA, Apple Silicon, AMD, Intel GPU,
  and CPU are auto-selected at runtime.
- **OpenAI-compatible.** Models run as drop-in replacements for the OpenAI
  API. The optional `gateway` command exposes every running model on a
  single port.
- **No build chain.** Backend binaries are downloaded for you;
  `llmconfig install <llama|sd|whisper>` is a one-shot. Optional:
  `llmconfig install ik_llama` builds the
  [ik_llama.cpp](https://github.com/ikawrakow/ik_llama.cpp) fork from
  source for SOTA quants and faster CPU / MoE inference.

## Install

Linux / macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/kiliczsh/llmconfig/refs/heads/main/install.sh | bash
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/kiliczsh/llmconfig/refs/heads/main/install.ps1 | iex
```

Or via Go:

```bash
go install github.com/kiliczsh/llmconfig@latest
```

Or build from source:

```bash
git clone https://github.com/kiliczsh/llmconfig
cd llmconfig
go build -o llmconfig .
```

Requires **Go 1.26+**.

## Quick Start

```bash
# 1. Install the llama.cpp binary (CUDA / Metal / CPU build auto-detected)
llmconfig install llama

# 2. Create a config from a built-in template (use `=`, not a space)
llmconfig init --template=gemma

# 3. Start the model
llmconfig up gemma

# 4. Send a request — OpenAI-compatible
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gemma","messages":[{"role":"user","content":"Hello!"}]}'

# 5. Stop
llmconfig down
```

For image generation or speech recognition, swap step 1 for `install sd`
or `install whisper` and pick a matching template.

## Documentation

| Page | What you'll find |
|------|------------------|
| [docs/reference.md](docs/reference.md) | Full reference — commands, config fields, hardware profiles, env vars, API |
| [docs/templates.md](docs/templates.md) | All 18 built-in templates with model details and recommended sizes |
| [CHANGELOG.md](CHANGELOG.md) | Release history |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Project layout, build, adding a template, sending a PR |

## Common commands

A handful of commands you'll reach for most often. The full list (with flags) is in [docs/reference.md → Commands](docs/reference.md#commands).

```bash
llmconfig up <name>          # start a model
llmconfig down [name]        # stop (interactive picker if multiple)
llmconfig ps                 # list running models
llmconfig logs <name> -f     # tail logs
llmconfig models             # list configured models
llmconfig init --template    # create a config from a template
llmconfig gateway            # unified API for every running model
llmconfig hardware           # show detected GPU / RAM / VRAM
```

## Contributing

Bug reports, new templates, and PRs are all welcome. Start with
[CONTRIBUTING.md](CONTRIBUTING.md) for the build and template authoring
guide. File issues with the [issue templates](.github/ISSUE_TEMPLATE/).

## License

[MIT](LICENSE)
