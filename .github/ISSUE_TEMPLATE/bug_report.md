---
name: Bug report
about: Report something that's broken or behaving unexpectedly
title: "[bug] "
labels: bug
---

## What happened

<!-- A clear, concise description of what went wrong. -->

## What you expected

<!-- What you thought would happen instead. -->

## Reproduction

<!--
Minimal steps to reproduce. Include the exact commands you ran.
If the bug involves a config, paste the relevant YAML (redact tokens / paths).
-->

```bash
# commands you ran
llmconfig ...
```

```yaml
# config (~/.llmconfig/configs/<name>.yaml), if relevant
```

## Logs

<!--
Paste the output of:
  llmconfig logs <name>            # or `llmconfig logs <name> -n 200`
  llmconfig --verbose <command>    # for verbose output
-->

```
<paste here>
```

## Environment

- **llmconfig version:** <!-- output of `llmconfig version` -->
- **Backend:** <!-- llama / sd / whisper -->
- **Backend binary version:** <!-- output of `llmconfig llama --version` (or sd / whisper) -->
- **OS:** <!-- e.g. Ubuntu 22.04 / macOS 14.4 / Windows 11 -->
- **Architecture:** <!-- x86_64 / arm64 -->
- **Hardware:** <!-- output of `llmconfig hardware` (CPU, GPU, VRAM, RAM) -->

## Additional context

<!-- Anything else worth knowing — recent changes, related issues, screenshots. -->
