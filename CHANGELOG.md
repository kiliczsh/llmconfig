# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

_Nothing yet._

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
