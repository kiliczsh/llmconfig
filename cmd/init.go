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
	"codellama": "TheBloke/CodeLlama-13B-Instruct-GGUF",
	"mistral":   "TheBloke/Mistral-7B-Instruct-v0.2-GGUF",
	"llama3":    "bartowski/Meta-Llama-3.1-8B-Instruct-GGUF",
	"deepseek":  "bartowski/DeepSeek-R1-Distill-Qwen-7B-GGUF",
	"phi4":      "bartowski/phi-4-GGUF",
	"gemma":     "bartowski/google_gemma-4-E2B-it-GGUF",
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

			// Pre-fill repo from --template
			if flagTemplate != "" && flagFrom == "" {
				if repo, ok := builtinTemplates[flagTemplate]; ok {
					flagFrom = repo
				}
			}

			// Fields to collect
			var (
				name         string
				repo         string
				file         string
				port         string = "8080"
				mode         string = "server"
				systemPrompt string
			)

			if len(args) > 0 {
				name = args[0]
			}
			if flagFrom != "" {
				repo = flagFrom
			}

			// Build form
			fields := []huh.Field{}

			if name == "" {
				fields = append(fields, huh.NewInput().
					Title("Model name").
					Description("Used in CLI commands: llamaconfig up <name>").
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
					Description("Only used in server mode").
					Placeholder("8080").
					Value(&port),

				huh.NewText().
					Title("System prompt").
					Description("Optional. Leave blank to skip.").
					Placeholder("You are a helpful assistant.").
					Value(&systemPrompt),
			)

			form := huh.NewForm(huh.NewGroup(fields...))
			if err := form.Run(); err != nil {
				return err
			}

			// Validate name
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("name cannot be empty")
			}
			repo = strings.TrimSpace(repo)
			if repo == "" {
				return fmt.Errorf("repo cannot be empty")
			}

			// Pick file from repo
			if file == "" {
				p.Info("listing files in %s...", repo)
				files, err := downloader.ListRepoFiles(repo, token)
				if err != nil {
					p.Warn("could not list repo files: %v — you can set model.file manually", err)
				} else {
					var ggufFiles []huh.Option[string]
					for _, f := range files {
						if strings.HasSuffix(f.Filename, ".gguf") && !strings.Contains(f.Filename, "mmproj") {
							ggufFiles = append(ggufFiles, huh.NewOption(f.Filename, f.Filename))
						}
					}
					if len(ggufFiles) > 0 {
						selectForm := huh.NewForm(huh.NewGroup(
							huh.NewSelect[string]().
								Title("Select GGUF file").
								Options(ggufFiles...).
								Value(&file),
						))
						if err := selectForm.Run(); err != nil {
							return err
						}
					}
				}
			}
			if file == "" {
				file = "<set model.file>"
			}

			// Determine output path
			outPath := flagOutput
			if outPath == "" {
				outPath = filepath.Join(appCtx.ConfigDir, name+".yaml")
			}

			if _, err := os.Stat(outPath); err == nil {
				var overwrite bool
				huh.NewForm(huh.NewGroup(
					huh.NewConfirm().
						Title(fmt.Sprintf("Config %q already exists. Overwrite?", outPath)).
						Value(&overwrite),
				)).Run()
				if !overwrite {
					p.Info("cancelled")
					return nil
				}
			}

			// Write config
			portNum := "8080"
			if port != "" {
				portNum = port
			}

			var sb strings.Builder
			fmt.Fprintf(&sb, "version: 1\n")
			fmt.Fprintf(&sb, "name: %s\n", name)
			fmt.Fprintf(&sb, "\nmode: %s\n", mode)
			fmt.Fprintf(&sb, "\nmodel:\n")
			fmt.Fprintf(&sb, "  source: huggingface\n")
			fmt.Fprintf(&sb, "  repo: %s\n", repo)
			fmt.Fprintf(&sb, "  file: %s\n", file)
			fmt.Fprintf(&sb, "\nserver:\n")
			fmt.Fprintf(&sb, "  port: %s\n", portNum)
			if systemPrompt != "" {
				fmt.Fprintf(&sb, "\nchat:\n")
				fmt.Fprintf(&sb, "  system_prompt: |\n")
				for _, line := range strings.Split(systemPrompt, "\n") {
					fmt.Fprintf(&sb, "    %s\n", line)
				}
			}

			if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
				return fmt.Errorf("init: write config: %w", err)
			}

			p.Success("config created: %s", outPath)
			p.Info("next: llamaconfig validate %s && llamaconfig up %s", name, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagFrom, "from", "", "pre-fill from a HuggingFace repo")
	cmd.Flags().StringVar(&flagTemplate, "template", "", "start from a built-in template: codellama | mistral | llama3 | deepseek | phi4 | gemma")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "write config to a specific path")
	return cmd
}
