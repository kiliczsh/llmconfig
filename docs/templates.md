# Built-in Templates

llmconfig ships with 18 ready-to-use templates embedded in the binary.
Each template is a YAML config tuned to a specific model with sensible
defaults and commented alternatives for different VRAM budgets.

## Using a template

```bash
llmconfig init --template               # interactive picker (no value)
llmconfig init --template=gemma         # skip picker, use this template
llmconfig init my-name --template=llama # custom config name + template
```

> **Note:** Pass template values with `--template=<name>` (with `=`).
> The bare `--template` form opens the picker by design — a
> space-separated value would be parsed as the positional config name
> instead.

The resulting config is written to
`~/.llmconfig/configs/<name>.llmc` (or
`%USERPROFILE%\.llmconfig\configs\<name>.llmc` on Windows). The default
config name matches the template name unless you override it with a
positional argument. Files are YAML inside; the `.llmc` extension is
the canonical filename for an llmconfig config.

Sizes below are the recommended quantization for ~16 GB VRAM; each
template ships with commented alternatives for other VRAM budgets.
Open the `.llmc` file after `init` to see them.

---

## Text — `llama` backend

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

## Image — `sd` backend

| Template | Model | Notes |
|----------|-------|-------|
| `sd` | Stable Diffusion 1.5 (RunwayML) | Classic checkpoint, 512×512 |
| `flux-schnell` | Black Forest Labs FLUX.1 Schnell | Distilled, 4 steps, 1024×1024 |
| `flux-dev` | Black Forest Labs FLUX.1 Dev | Higher quality, 20 steps, 1024×1024 |

## Speech — `whisper` backend

| Template | Model | Notes |
|----------|-------|-------|
| `whisper` | OpenAI Whisper (ggml) | Defaults to `base`; pick `medium`, `large-v3-turbo`, etc. |

---

## Adding your own template

Templates are plain YAML files in `templates/` (with the `.llmc`
extension), embedded into the binary via `go:embed`. To add one:

1. Create `templates/<name>.llmc` based on the closest existing template
   (`gemma.llmc` for chat, `flux-schnell.llmc` for image, `whisper.llmc`
   for speech).
2. Start with a 2–3 line header comment describing the model.
3. Use the canonical `hardware_profiles` ordering: `nvidia →
   apple_silicon → cpu`.
4. Rebuild: `go build -o llmconfig .` — the new template is picked up
   automatically.

See [CONTRIBUTING.md → Adding a new model template](../CONTRIBUTING.md#adding-a-new-model-template)
for the full step-by-step guide.
