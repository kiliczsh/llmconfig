package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/kiliczsh/llamaconfig/internal/downloader"
	"github.com/spf13/cobra"
)

var builtinTemplates = map[string]string{
	// llama
	"codellama": "TheBloke/CodeLlama-13B-Instruct-GGUF",
	"mistral":   "TheBloke/Mistral-7B-Instruct-v0.2-GGUF",
	"llama3":    "bartowski/Meta-Llama-3.1-8B-Instruct-GGUF",
	"deepseek":  "bartowski/DeepSeek-R1-Distill-Qwen-7B-GGUF",
	"phi4":      "bartowski/phi-4-GGUF",
	"gemma":     "bartowski/google_gemma-4-E2B-it-GGUF",
	// sd
	"sd15":         "runwayml/stable-diffusion-v1-5",
	"flux-schnell": "city96/FLUX.1-schnell-gguf",
	"flux-dev":     "city96/FLUX.1-dev-gguf",
	// whisper
	"whisper-base":  "ggerganov/whisper.cpp",
	"whisper-turbo": "ggerganov/whisper.cpp",
}

func newInitCmd() *cobra.Command {
	var flagFrom string
	var flagTemplate string
	var flagOutput string

	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Create a new model config interactively",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer
			token := resolveToken("")

			var name string
			if len(args) > 0 {
				name = args[0]
			}

			// Step 1: backend
			backend := "llama"
			if err := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().
					Title("Backend").
					Options(
						huh.NewOption("llama  — text generation (llama.cpp)", "llama"),
						huh.NewOption("sd     — image generation (stable-diffusion.cpp)", "sd"),
						huh.NewOption("whisper — speech recognition (whisper.cpp)", "whisper"),
					).
					Value(&backend),
			)).Run(); err != nil {
				return err
			}

			switch backend {
			case "sd":
				return initSD(name, flagFrom, flagTemplate, flagOutput, appCtx.ConfigDir, p)
			case "whisper":
				return initWhisper(name, flagFrom, flagTemplate, flagOutput, appCtx.ConfigDir, p)
			default:
				return initLlama(name, flagFrom, flagTemplate, flagOutput, appCtx.ConfigDir, token, p)
			}
		},
	}

	cmd.Flags().StringVar(&flagFrom, "from", "", "pre-fill from a HuggingFace repo or URL")
	cmd.Flags().StringVar(&flagTemplate, "template", "", "built-in template: codellama|mistral|llama3|deepseek|phi4|gemma|sd15|flux-schnell|whisper-base|whisper-turbo")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "write config to a specific path")
	return cmd
}

func initLlama(name, flagFrom, flagTemplate, flagOutput, configDir, token string, p interface {
	Info(string, ...any)
	Warn(string, ...any)
	Success(string, ...any)
}) error {
	if flagTemplate != "" && flagFrom == "" {
		if repo, ok := builtinTemplates[flagTemplate]; ok {
			flagFrom = repo
		}
	}

	var (
		repo         string
		file         string
		port         = "8080"
		mode         = "server"
		systemPrompt string
	)
	if flagFrom != "" {
		repo = flagFrom
	}

	fields := []huh.Field{}
	if name == "" {
		fields = append(fields, huh.NewInput().
			Title("Model name").
			Placeholder("my-model").
			Value(&name))
	}
	if repo == "" {
		fields = append(fields, huh.NewInput().
			Title("HuggingFace repo").
			Description("e.g. bartowski/google_gemma-4-E2B-it-GGUF").
			Placeholder("user/repo-GGUF").
			Value(&repo))
	}
	fields = append(fields,
		huh.NewSelect[string]().
			Title("Mode").
			Options(
				huh.NewOption("server (OpenAI-compatible API)", "server"),
				huh.NewOption("interactive (llama-cli terminal chat)", "interactive"),
			).
			Value(&mode),
		huh.NewInput().
			Title("Server port").
			Placeholder("8080").
			Value(&port),
		huh.NewText().
			Title("System prompt").
			Description("Optional. Leave blank to skip.").
			Placeholder("You are a helpful assistant.").
			Value(&systemPrompt),
	)

	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("repo cannot be empty")
	}

	if file == "" {
		p.Info("listing files in %s...", repo)
		files, err := downloader.ListRepoFiles(repo, token)
		if err != nil {
			p.Warn("could not list repo files: %v — set model.file manually", err)
		} else {
			var opts []huh.Option[string]
			for _, f := range files {
				if strings.HasSuffix(f.Filename, ".gguf") && !strings.Contains(f.Filename, "mmproj") {
					opts = append(opts, huh.NewOption(f.Filename, f.Filename))
				}
			}
			if len(opts) > 0 {
				if err := huh.NewForm(huh.NewGroup(
					huh.NewSelect[string]().Title("Select GGUF file").Options(opts...).Value(&file),
				)).Run(); err != nil {
					return err
				}
			}
		}
	}
	if file == "" {
		file = "<set model.file>"
	}
	if port == "" {
		port = "8080"
	}

	outPath := resolveOutPath(flagOutput, configDir, name)
	if cancelled, err := confirmOverwrite(outPath, p); err != nil || cancelled {
		return err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "version: 1\nname: %s\n\nbackend: llama\n\nmode: %s\n\nmodel:\n  source: huggingface\n  repo: %s\n  file: %s\n  download:\n    resume: true\n\nserver:\n  port: %s\n", name, mode, repo, file, port)
	if systemPrompt != "" {
		fmt.Fprintf(&sb, "\nchat:\n  system_prompt: |\n")
		for _, line := range strings.Split(systemPrompt, "\n") {
			fmt.Fprintf(&sb, "    %s\n", line)
		}
	}

	return writeConfig(outPath, sb.String(), name, p)
}

func initSD(name, flagFrom, flagTemplate, flagOutput, configDir string, p interface {
	Info(string, ...any)
	Warn(string, ...any)
	Success(string, ...any)
}) error {
	if flagTemplate != "" && flagFrom == "" {
		if repo, ok := builtinTemplates[flagTemplate]; ok {
			flagFrom = repo
		}
	}

	var (
		repo           string
		file           string
		port           = "8090"
		width          = "512"
		height         = "512"
		steps          = "20"
		cfgScale       = "7.0"
		samplingMethod = "euler_a"
	)
	if flagFrom != "" {
		repo = flagFrom
	}

	fields := []huh.Field{}
	if name == "" {
		fields = append(fields, huh.NewInput().
			Title("Model name").
			Placeholder("my-sd-model").
			Value(&name))
	}

	sourceType := "huggingface"
	fields = append(fields,
		huh.NewSelect[string]().
			Title("Model source").
			Options(
				huh.NewOption("HuggingFace repo", "huggingface"),
				huh.NewOption("Direct URL", "url"),
			).
			Value(&sourceType),
	)

	if repo == "" {
		fields = append(fields, huh.NewInput().
			Title("HuggingFace repo or URL").
			Description("e.g. city96/FLUX.1-schnell-gguf").
			Placeholder("user/repo  or  https://...").
			Value(&repo))
	}

	fields = append(fields,
		huh.NewInput().Title("Server port").Placeholder("8090").Value(&port),
		huh.NewSelect[string]().
			Title("Image size").
			Options(
				huh.NewOption("512×512  (SD 1.x)", "512"),
				huh.NewOption("768×768  (SD 2.x)", "768"),
				huh.NewOption("1024×1024  (SDXL / FLUX)", "1024"),
			).
			Value(&width),
		huh.NewInput().Title("Steps").Placeholder("20").Value(&steps),
		huh.NewInput().Title("CFG scale").Placeholder("7.0").Value(&cfgScale),
		huh.NewSelect[string]().
			Title("Sampling method").
			Options(
				huh.NewOption("euler_a", "euler_a"),
				huh.NewOption("euler", "euler"),
				huh.NewOption("dpm++2m", "dpm++2m"),
				huh.NewOption("lcm", "lcm"),
			).
			Value(&samplingMethod),
	)

	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	height = width // square by default

	if port == "" {
		port = "8090"
	}

	outPath := resolveOutPath(flagOutput, configDir, name)
	if cancelled, err := confirmOverwrite(outPath, p); err != nil || cancelled {
		return err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "version: 1\nname: %s\n\nbackend: sd\n\nmodel:\n", name)
	if sourceType == "url" {
		fmt.Fprintf(&sb, "  source: url\n  url: %s\n", repo)
		if file != "" {
			fmt.Fprintf(&sb, "  file: %s\n", file)
		}
	} else {
		fmt.Fprintf(&sb, "  source: huggingface\n  repo: %s\n", repo)
		if file != "" {
			fmt.Fprintf(&sb, "  file: %s\n", file)
		}
	}
	fmt.Fprintf(&sb, "  download:\n    resume: true\n    connections: 4\n")
	fmt.Fprintf(&sb, "\nmode: server\n\nserver:\n  host: 127.0.0.1\n  port: %s\n", port)
	fmt.Fprintf(&sb, "\nsd:\n  width: %s\n  height: %s\n  steps: %s\n  cfg_scale: %s\n  sampling_method: %s\n  seed: -1\n", width, height, steps, cfgScale, samplingMethod)

	return writeConfig(outPath, sb.String(), name, p)
}

func initWhisper(name, flagFrom, flagTemplate, flagOutput, configDir string, p interface {
	Info(string, ...any)
	Warn(string, ...any)
	Success(string, ...any)
}) error {
	if flagTemplate != "" && flagFrom == "" {
		switch flagTemplate {
		case "whisper-turbo":
			flagFrom = "ggml-large-v3-turbo.bin"
		case "whisper-base":
			flagFrom = "ggml-base.bin"
		}
	}

	var (
		file     = "ggml-base.bin"
		port     = "8082"
		language = "auto"
		task     = "transcribe"
	)
	if flagFrom != "" {
		file = flagFrom
	}

	fields := []huh.Field{}
	if name == "" {
		fields = append(fields, huh.NewInput().
			Title("Model name").
			Placeholder("whisper-base").
			Value(&name))
	}

	fields = append(fields,
		huh.NewSelect[string]().
			Title("Model size").
			Options(
				huh.NewOption("tiny    (~75 MB)", "ggml-tiny.bin"),
				huh.NewOption("base    (~142 MB)", "ggml-base.bin"),
				huh.NewOption("small   (~466 MB)", "ggml-small.bin"),
				huh.NewOption("medium  (~1.5 GB)", "ggml-medium.bin"),
				huh.NewOption("large-v3-turbo  (~800 MB) — recommended", "ggml-large-v3-turbo.bin"),
				huh.NewOption("large-v3  (~2.9 GB)", "ggml-large-v3.bin"),
			).
			Value(&file),
		huh.NewInput().Title("Server port").Placeholder("8082").Value(&port),
		huh.NewSelect[string]().
			Title("Language").
			Options(
				huh.NewOption("auto-detect", "auto"),
				huh.NewOption("Turkish (tr)", "tr"),
				huh.NewOption("English (en)", "en"),
			).
			Value(&language),
		huh.NewSelect[string]().
			Title("Task").
			Options(
				huh.NewOption("transcribe (keep original language)", "transcribe"),
				huh.NewOption("translate (to English)", "translate"),
			).
			Value(&task),
	)

	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if port == "" {
		port = "8082"
	}

	outPath := resolveOutPath(flagOutput, configDir, name)
	if cancelled, err := confirmOverwrite(outPath, p); err != nil || cancelled {
		return err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "version: 1\nname: %s\n\nbackend: whisper\n\nmodel:\n  source: huggingface\n  repo: ggerganov/whisper.cpp\n  file: %s\n  download:\n    resume: true\n\nmode: server\n\nserver:\n  host: 127.0.0.1\n  port: %s\n\nwhisper:\n  language: %s\n  task: %s\n  beam_size: 5\n  best_of: 5\n  vad: true\n  vad_threshold: 0.5\n  processors: 1\n", name, file, port, language, task)

	return writeConfig(outPath, sb.String(), name, p)
}

func resolveOutPath(flagOutput, configDir, name string) string {
	if flagOutput != "" {
		return flagOutput
	}
	return filepath.Join(configDir, name+".yaml")
}

func confirmOverwrite(outPath string, p interface{ Info(string, ...any) }) (cancelled bool, err error) {
	if _, err := os.Stat(outPath); err != nil {
		return false, nil
	}
	var overwrite bool
	if err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(fmt.Sprintf("Config %q already exists. Overwrite?", outPath)).
			Value(&overwrite),
	)).Run(); err != nil {
		return false, err
	}
	if !overwrite {
		p.Info("cancelled")
		return true, nil
	}
	return false, nil
}

func writeConfig(outPath, content, name string, p interface {
	Success(string, ...any)
	Info(string, ...any)
}) error {
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("init: write config: %w", err)
	}
	p.Success("config created: %s", outPath)
	p.Info("next: llamaconfig validate %s && llamaconfig up %s", name, name)
	return nil
}
