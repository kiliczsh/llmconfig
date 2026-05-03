<!--
Thanks for contributing to llmconfig!

Before opening this PR, please confirm:
- `go build ./...` passes locally
- you have read CONTRIBUTING.md
- the change is focused (one feature / one fix per PR)
-->

## Summary

<!-- What does this PR change and why? One or two sentences is plenty. -->

## Related issue

<!-- e.g. "Closes #123" or "Refs #123". Delete this section if there is no related issue. -->

## Type of change

- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change (existing config files, command flags, or on-disk layout)
- [ ] New template (`templates/<name>.yaml`)
- [ ] Documentation only

## Testing

<!--
Describe how you verified this change. Examples:
- `go build ./...` passes
- `llmconfig validate <name>` against the affected templates
- Ran `llmconfig up <name>` end-to-end on Linux/CUDA
-->

## Checklist

- [ ] Code builds (`go build ./...`)
- [ ] Touched commands run without error in a quick smoke test
- [ ] If a flag, command, or config field was added/changed, `README.md` and `DOCS.md` are updated
- [ ] If a template was added, it has a header comment describing the model and target VRAM
- [ ] `CHANGELOG.md` has an `[Unreleased]` entry for user-visible changes
