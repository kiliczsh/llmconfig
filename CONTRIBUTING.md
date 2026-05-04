# Contributing to llmconfig

Thanks for taking the time to contribute! This guide explains how to set up
the project, how it's organized, and how to get a change merged.

If you're filing a bug or suggesting a feature, please use the [issue
templates](.github/ISSUE_TEMPLATE/) — they ask for the information that
makes triage fast.

---

## Prerequisites

- **Go 1.26+** — see `go.mod` for the exact version.
- **Git**.
- A C/C++ runtime is *not* required to develop on llmconfig itself; the
  inference backends (`llama.cpp`, `stable-diffusion.cpp`, `whisper.cpp`)
  are downloaded as prebuilt binaries by `llmconfig install <backend>`.

---

## Getting set up

```bash
git clone https://github.com/kiliczsh/llmconfig
cd llmconfig
go build -o llmconfig .

./llmconfig version
./llmconfig --help
```

For day-to-day work it's convenient to install your local build onto your
`PATH`:

```bash
go install .
which llmconfig
```

llmconfig stores all runtime state under `~/.llmconfig/` (or
`%USERPROFILE%\.llmconfig\` on Windows). Set `LLMCONFIG_CONFIG_DIR` to
point at a throw-away directory while developing so you don't pollute your
real config:

```bash
export LLMCONFIG_CONFIG_DIR=/tmp/llmconfig-dev
```

---

## Project layout

```
llmconfig/
├── cmd/                 # cobra commands — one file per command (up.go, down.go, ...)
│   └── root.go          # root command, AppContext, command registration
├── internal/            # internal packages (not importable from outside)
│   ├── config/          # YAML parsing, defaults, validation
│   ├── dirs/            # resolves ~/.llmconfig and its subdirectories
│   ├── downloader/      # HuggingFace / URL download with resume
│   ├── hardware/        # CPU / GPU / RAM detection
│   ├── output/          # printer (color, verbose, no-color)
│   ├── runner/          # process lifecycle (start, health, stop)
│   └── state/           # state.json: which models are running with which PID
├── pkg/                 # backend wrappers, importable
│   ├── llamacpp/
│   ├── stablediffusioncpp/
│   └── whispercpp/
├── templates/           # built-in YAML templates, embedded via go:embed
│   └── embed.go         # //go:embed *.llmc
├── references/          # reference configs documenting every backend flag
├── .github/             # workflows, issue + PR templates
├── docs/                # long-form documentation
│   ├── README.md        # docs index
│   ├── reference.md     # full reference (commands, config, env vars, API)
│   └── templates.md     # built-in template catalogue
├── README.md            # short overview + quick start
└── main.go              # entrypoint — calls cmd.Execute()
```

A few things worth knowing:

- **Every command lives in its own file** under `cmd/`, named after the
  command (`up.go`, `gateway.go`, `state.go`, ...). Adding a new command
  means: create a `new<Name>Cmd()` function and register it in
  `cmd/root.go` inside the `rootCmd.AddCommand(...)` block.
- **Templates are embedded** via `templates/embed.go`. New `.llmc` files
  in `templates/` are picked up automatically at the next `go build` —
  no code change required. (Configs are still YAML inside; the `.llmc`
  extension is just the canonical filename for an llmconfig config.)
- **Defaults are applied at load time** by `internal/config`, so templates
  can stay minimal. The full set of fields lives in [docs/reference.md](docs/reference.md#config-file-reference).

---

## Building and running

```bash
go build ./...                         # compile everything
go vet ./...                           # static checks
gofmt -l .                             # list any unformatted files
gofmt -w .                             # format in place

go build -o llmconfig . && ./llmconfig hardware
```

There are no unit tests yet. If your change is amenable to testing —
especially in `internal/config` (parsing, defaults), `internal/state`
(locking, prune), or `internal/downloader` — please add `*_test.go` files
alongside the code.

For end-to-end smoke tests, the cheapest path is:

```bash
./llmconfig install llama
./llmconfig init --template=gemma
./llmconfig validate gemma
./llmconfig up gemma
./llmconfig ps
./llmconfig down gemma
```

---

## Adding a new model template

Templates are the easiest way to contribute. They're plain YAML files in
`templates/` (with the `.llmc` extension) and ship in the binary via
`go:embed`.

1. Pick a short, lowercase name: `templates/<name>.llmc`. The name doubles
   as the user-visible `--template` value and as the default config name.
2. Copy the closest existing template (e.g. `gemma.llmc` for a llama-based
   chat model, `flux-schnell.llmc` for an image model, `whisper.llmc` for
   speech).
3. Start the file with a 2–3 line header comment that tells the user what
   they're configuring:

   ```yaml
   # Mistral 7B Instruct v0.3 — general-purpose chat (~7B params).
   # Source: bartowski/Mistral-7B-Instruct-v0.3-GGUF
   # Recommended: 16 GB VRAM at Q4_K_M.
   ```

4. Set `name:` to the template name (the user can override at `init` time
   with `llmconfig init <custom-name> --template <name>`).
5. Pick a sensible default `file:` quantization for ~16 GB VRAM and leave
   2–3 commented alternatives for higher- and lower-VRAM users:

   ```yaml
   model:
     file: <recommended>.gguf      # 8.7GB — recommended for 16GB VRAM
     # file: <higher-quality>.gguf # 11.7GB — better quality, tighter on 16GB
     # file: <smaller>.gguf        # 4.9GB — fits 8GB VRAM
   ```

6. Use the canonical `hardware_profiles` ordering: `nvidia → apple_silicon
   → cpu`. Other profiles (`amd`, `intel_gpu`) can be added if relevant.
7. Verify it loads without warnings:

   ```bash
   go build -o llmconfig .
   ./llmconfig init --template=<name>     # use `=` when passing a value
   ./llmconfig validate <name>
   ./llmconfig inspect <name> --dry-run
   ```

8. Add a row to the **Templates** table in `README.md`.

---

## Commit and PR conventions

### Commit messages

Match the existing style in `git log`:

```
feat(scope): short description
fix(scope): short description
docs: short description
refactor(scope): short description
chore: short description
```

Common scopes: `init`, `up`, `gateway`, `downloader`, `templates`,
`config`, `state`, `ux`, `auth`, `bench`, `sd`.

Keep the subject line under ~70 characters. Use the body to explain *why*
the change was needed when it isn't obvious from the diff.

### Pull requests

1. Fork the repo and create a branch off `main`.
2. Keep PRs focused — one feature or one fix per PR is much easier to
   review than a large bundle.
3. Make sure `go build ./...` and `go vet ./...` both pass.
4. Update `README.md` and/or `docs/reference.md` if you added or changed a command,
   flag, or config field.
5. Add an entry under `[Unreleased]` in `CHANGELOG.md` for any
   user-visible change.
6. Open the PR using the [pull request
   template](.github/PULL_REQUEST_TEMPLATE.md). Link any related issue
   with `Closes #N`.

---

## Releases

Releases are cut by pushing a `v*` tag — the
[`.github/workflows/release.yml`](.github/workflows/release.yml) workflow
runs `goreleaser` on tag push to build cross-platform binaries and create a
GitHub Release.

Before tagging:
1. Move everything from `[Unreleased]` in `CHANGELOG.md` into a new
   versioned section with the release date.
2. Push the tag: `git tag v0.x.0 && git push origin v0.x.0`.

---

## Reporting security issues

Please **don't** open a public GitHub issue for security problems. Email
the maintainer directly (see the email associated with recent commits)
with a description and reproduction steps.

---

## Code of conduct

Be kind, be specific, assume good faith. Report any conduct concerns to
the maintainer via email.
