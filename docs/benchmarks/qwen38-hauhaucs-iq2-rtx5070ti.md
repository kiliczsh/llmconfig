# HauhauCS Qwen3.8 IQ2_M on RTX 5070 Ti

Measured on 2026-10-30 with a GeForce RTX 5070 Ti (16,303 MiB),
Windows WDDM, NVIDIA driver 591.86, and llama.cpp build 10964
(`b29c606e2`, version 0.4.1-dev). No other inference server was running.

Model: [HauhauCS Aggressive IQ2_M](https://huggingface.co/HauhauCS/Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-MTP-GGUF).
The 10,319,906,944-byte download was verified against the Hugging Face
SHA-256: `5e14825174e02733b873cc01358787d570b648dd530c061623764aba743512ce`.

## Recommended context profile

The `qwen38-hauhaucs-iq2` template uses 131,072 context tokens, Q8_0 K/V,
flash attention, full GPU layer offload, one server slot, batch 2048,
micro-batch 512, and no speculative decoding. This favors context capacity
while retaining about 1.15 GiB of free VRAM on the tested machine.

| Context tokens | Embedded MTP depth | Generation tok/s | Prompt tok/s | Minimum free VRAM (MiB) |
|---:|---:|---:|---:|---:|
| 65,536 | Off | 56.63 | 1,587 | 3,672 |
| 98,304 | Off | 56.65 | 1,588 | 2,424 |
| 131,072 | Off | 56.58 | 1,588 | 1,176 |
| 163,840 | Off | 56.58 | 1,577 | 54 |
| 98,304 | 2 | 90.48 | 1,510 | 1,354 |

The 160K setting is unsuitable as a default despite its short-prompt speed:
54 MiB is too little margin for other GPU activity or larger workloads.
For a speed alternative, lower context to 98,304 and enable the template's
commented `draft-mtp` block with `draft_n: 2`. FastMTP was not tested.

At 128K, increasing micro-batch from 512 to 1024 produced 1,614.58 prompt
tok/s and 56.68 generation tok/s, with only 962 MiB free. The roughly
1.7% prompt-speed gain did not justify the reduced memory margin, so the
template keeps micro-batch 512.

## Actual long-prompt validation

At 131,072 configured context, a fresh 120,000-token prompt completed
without truncation or prompt-cache reuse:

- Prompt processing: 142.10 seconds, 844.46 tok/s.
- Generation: 128 tokens, 31.48 tok/s.
- Total request: 146.28 seconds.
- Minimum free VRAM: 1,176 MiB.

Configured capacity does not imply the same speed at a full context:
generation was 56.58 tok/s at a 2K prompt and 31.48 tok/s at a 120K prompt.

## Method and limits

Each short-prompt result is the median of two HTTP `/completion` requests
after a warmup, with 2,048 input tokens and 512 generated tokens. Requests
used `cache_prompt=false`; response timings confirmed `cache_n=0` and
`prompt_n=2048`. Sampling was temperature 1.0, top-k 20, top-p 0.95,
min-p 0, repeat penalty 1.0, seed 42, and `ignore_eos=true` to keep the
generation length fixed. Synthetic numbered records supplied the input.
GPU memory was sampled every 0.5 seconds with `nvidia-smi`.

These are throughput and capacity measurements, not accuracy, reasoning,
retrieval-quality, or concurrent-client benchmarks. Other prompts, desktop
GPU use, driver versions, and runtime builds can change the results.

## llmconfig integration

The integration check exposed an argument-generation bug: `parallel: 1`
was omitted for the stock backend, which auto-selected four slots and
reduced free VRAM to about 730 MiB. The runner now forwards the resolved
positive parallel value for all llama-compatible backends, with a
regression test covering the single-slot default.

After restarting with the rebuilt CLI, `/props` confirmed one slot and
131,072 context tokens; `/health` returned `ok`, and a chat arithmetic
smoke check returned the expected answer. Free VRAM was 1,182 MiB.
Use the rebuilt repository CLI for this fix; an older installed CLI can
still omit the parallel flag.
