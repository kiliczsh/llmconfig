# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `.llmc` is now the canonical extension for config files. New
  templates, `init`, `add`, `pull`, and archive imports all write
  `.llmc`. Built-in templates were renamed (e.g. `templates/gemma.yaml`
  → `templates/gemma.llmc`); files are still YAML inside.
- Auto-migration on first run: existing `~/.llmconfig/configs/*.yaml`
  files are renamed to `*.llmc` with a `*.yaml.bak` backup left
  alongside, gated by a marker file so the scan only runs once.
  Conflicts (both extensions for the same name) are skipped with a
  warning.
- `internal/config` exports `ConfigPath`, `FindConfigInDir`,
  `ListConfigNames`, `ListConfigPaths`, `IsConfigFile`, and
  `TrimConfigExt` so cmd/ doesn't have to repeat extension-aware
  globs.

### Changed
- Loader (`internal/config.Load`) now searches for both `.llmc` and
  `.yaml`, with `.llmc` winning when both exist. Bare-name lookups via
  CLI args resolve transparently to whichever file is present.
- Archive bundles (`.llmcpkg`) now ship configs as
  `configs/<name>.llmc`. Reading still accepts legacy bundles whose
  inner config is `.yaml`; on import the file lands as `.llmc` on
  disk regardless.

- `llmconfig update` — self-update command. Downloads the latest
  release from GitHub, verifies its SHA256 against `checksums.txt`, and
  atomically replaces the running binary (the previous binary is kept
  as `<binary>.old`). Flags: `--check` (no install), `--version <tag>`
  (pin a specific release / downgrade), `--force` (reinstall on
  current). Updates the `llmc` alias binary in the same operation
  when present.

### Changed
- `llmconfig version --check` now points users at `llmconfig update`
  instead of the bootstrap install scripts.

## [1.1.0] - 2026-05-04

### Added
- `ik_llama` backend — drop-in alternative to `llama` that runs the
  [ikawrakow/ik_llama.cpp](https://github.com/ikawrakow/ik_llama.cpp)
  fork (SOTA quants, MLA, fused MoE). Set `backend: ik_llama` in any
  config; ik-only flags live under a new optional `ik_llama` block
  (`rtr`, `mla`, `fmoe`, `ser`, `cuda_graphs`).
- `llmconfig install ik_llama` — clones and cmake-builds ik_llama.cpp
  into `~/.llmconfig/cache/` and installs `llama-server` / `llama-cli`
  into `~/.llmconfig/bin/ik-llama/`. Supports `--backend cpu|cuda`,
  `--ref <tag|commit>`, `--jobs N`, `--verbose`, and `--file <archive>`
  (bring-your-own-binary). Build log lands at
  `~/.llmconfig/logs/install-ik-llama.log`.
- `llmconfig ik_llama` — status command (path, version) mirroring
  `llmconfig llama`.

## [1.0.0] - 2026-05-03

First stable release. The `1.0` line stabilizes the CLI surface
(commands, flags, config schema, on-disk layout) and ships with a
documented contributor workflow. Future breaking changes will go through
a deprecation cycle and a major version bump.

### Added
- `gateway` command — a unified OpenAI-compatible HTTP gateway that routes
  requests to the right running model based on the `"model"` field in the
  request body.
- `state prune` command — marks running entries with dead PIDs as stopped in
  the state file.
- Bare `--template` flag on `init` — running `llmconfig init --template`
  with no value opens an interactive template picker.
- Templates: `flux-schnell`, `flux-dev`, `phi4`, plus seven additional
  unsloth-based templates tuned for 16 GB VRAM. 18 built-in templates
  total.
- Embedded templates (`templates/*.yaml` via `go:embed`) — one source of
  truth for both the picker and `--template=<name>`.
- Interactive pickers and shell autocomplete for every command.
- `LICENSE` (MIT), `CONTRIBUTING.md`, `CHANGELOG.md`, and GitHub issue /
  pull-request templates.
- `docs/` folder with `reference.md` (full reference) and `templates.md`
  (template catalogue).

### Changed
- Cache directory and command renamed: the on-disk directory is now
  `~/.llmconfig/models/` and the management command is `llmconfig files`
  (previously `llmconfig cache`).
- `llmconfig` rebranded from `llamaconfig`. The shorter `llmc` alias works
  everywhere too.
- Documentation reorganized: README slimmed to fit a single screen,
  long-form docs moved into `docs/`. The reference list of templates was
  also corrected to match what actually ships.

### Fixed
- Downloader now writes to `<file>.tmp` and renames to the final filename
  on completion, so an interrupted download is never mistaken for a
  complete one.
- File handle is closed on the early-return path in the downloader.
- Gateway returns clearer errors when no models are running or when the
  requested model is not available.
- ESC binding cancels all `huh` forms cleanly (pickers and confirm prompts).
- Hardware-aware health check accepts `401` and `403` as "ready" for
  auth-protected backends.
- HuggingFace 401/403 errors point at the exact repo page; falls back to
  the `hf` CLI's on-disk token when env vars are unset.
- Concurrency and safety: state-file locking, signal handling, archive
  collision handling.

[Unreleased]: https://github.com/kiliczsh/llmconfig/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/kiliczsh/llmconfig/releases/tag/v1.0.0
