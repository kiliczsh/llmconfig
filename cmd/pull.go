package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/dirs"
	"github.com/kiliczsh/llamaconfig/internal/downloader"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	var flagFile string
	var flagQuant string
	var flagName string
	var flagNoConfig bool
	var flagToken string

	cmd := &cobra.Command{
		Use:   "pull <repo>",
		Short: "Download a model from HuggingFace",
		Example: `  llamaconfig pull bartowski/google_gemma-4-E2B-it-GGUF --quant Q4_K_M
  llamaconfig pull TheBloke/Mistral-7B-Instruct-v0.2-GGUF --file mistral-7b-instruct-v0.2.Q4_K_M.gguf`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := args[0]
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			token := resolveToken(flagToken)

			// Resolve filename
			file := flagFile
			if file == "" && flagQuant == "" {
				return fmt.Errorf("provide --file <filename> or --quant <Q4_K_M|Q5_K_M|...>")
			}
			if file == "" {
				p.Info("searching %s for %s...", repo, flagQuant)
				var size int64
				var err error
				file, size, err = downloader.FindGGUF(repo, flagQuant, token)
				if err != nil {
					return err
				}
				p.Info("found: %s (%s)", file, humanize.Bytes(uint64(size)))
			}

			cacheDir := dirs.CacheDir()
			destPath := filepath.Join(cacheDir, file)

			// Already cached?
			if info, err := os.Stat(destPath); err == nil {
				p.Success("already cached: %s (%s)", file, humanize.Bytes(uint64(info.Size())))
				if !flagNoConfig {
					return writeConfigIfNeeded(appCtx, repo, file, flagName)
				}
				return nil
			}

			req := &downloader.Request{
				Repo:        repo,
				File:        file,
				Token:       token,
				CacheDir:    cacheDir,
				Resume:      true,
				Connections: 4,
			}

			if err := runDownloadWithProgress(cmd.Context(), req, file); err != nil {
				return err
			}

			p.Success("downloaded: %s", destPath)

			if !flagNoConfig {
				return writeConfigIfNeeded(appCtx, repo, file, flagName)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagFile, "file", "", "specific GGUF file to download")
	cmd.Flags().StringVar(&flagQuant, "quant", "", "quantization shorthand: Q4_K_M | Q5_K_M | Q8_0 ...")
	cmd.Flags().StringVar(&flagName, "name", "", "model name (default: derived from repo)")
	cmd.Flags().BoolVar(&flagNoConfig, "no-config", false, "download only, do not create config file")
	cmd.Flags().StringVar(&flagToken, "token", "", "HuggingFace token for private repos")
	return cmd
}

// ── Progress TUI ──────────────────────────────────────────────────────────────

type dlProgressMsg struct{ downloaded, total int64 }
type dlDoneMsg struct{ err error }

type dlProgressModel struct {
	bar        progress.Model
	label      string
	downloaded int64
	total      int64
	done       bool
	err        error
}

func newDlProgressModel(label string) dlProgressModel {
	return dlProgressModel{
		bar:   progress.New(progress.WithDefaultGradient()),
		label: label,
	}
}

func (m dlProgressModel) Init() tea.Cmd { return nil }

func (m dlProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dlDoneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit

	case dlProgressMsg:
		m.downloaded = msg.downloaded
		m.total = msg.total
		if m.total > 0 {
			cmd := m.bar.SetPercent(float64(m.downloaded) / float64(m.total))
			return m, cmd
		}
		return m, nil

	case progress.FrameMsg:
		barModel, cmd := m.bar.Update(msg)
		m.bar = barModel.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m dlProgressModel) View() string {
	if m.done {
		if m.err != nil {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗ "+m.err.Error()) + "\n"
		}
		done := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		return done.Render(fmt.Sprintf("✓ %s (%s)", m.label, humanize.Bytes(uint64(m.downloaded)))) + "\n"
	}

	bar := m.bar.View()
	var speed string
	if m.total > 0 {
		speed = fmt.Sprintf("  %s / %s", humanize.Bytes(uint64(m.downloaded)), humanize.Bytes(uint64(m.total)))
	} else {
		speed = fmt.Sprintf("  %s", humanize.Bytes(uint64(m.downloaded)))
	}
	return fmt.Sprintf("Downloading %s\n%s%s\n", m.label, bar, speed)
}

func runDownloadWithProgress(ctx context.Context, req *downloader.Request, label string) error {
	prog := tea.NewProgram(newDlProgressModel(label))

	dl := downloader.New()
	go func() {
		_, err := dl.Download(ctx, req, func(downloaded, total int64) {
			prog.Send(dlProgressMsg{downloaded: downloaded, total: total})
		})
		prog.Send(dlDoneMsg{err: err})
	}()

	finalModel, err := prog.Run()
	if err != nil {
		return fmt.Errorf("progress UI error: %w", err)
	}

	m := finalModel.(dlProgressModel)
	return m.err
}

// ── Config helpers ─────────────────────────────────────────────────────────────

func resolveToken(flagToken string) string {
	if flagToken != "" {
		return flagToken
	}
	if t := os.Getenv("HUGGINGFACE_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("HF_TOKEN")
}

func writeConfigIfNeeded(appCtx *AppContext, repo, file, nameOverride string) error {
	name := nameOverride
	if name == "" {
		name = deriveName(repo, file)
	}

	configPath := filepath.Join(appCtx.ConfigDir, name+".yaml")
	if _, err := os.Stat(configPath); err == nil {
		return nil // already exists
	}

	cfg := &config.Config{
		Version: 1,
		Name:    name,
		Model: config.ModelSpec{
			Source: "huggingface",
			Repo:   repo,
			File:   file,
		},
	}
	config.ApplyDefaults(cfg)

	content := fmt.Sprintf("version: 1\nname: %s\n\nmodel:\n  source: huggingface\n  repo: %s\n  file: %s\n\nserver:\n  port: %d\n",
		cfg.Name, cfg.Model.Repo, cfg.Model.File, cfg.Server.Port)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("pull: write config: %w", err)
	}

	appCtx.Printer.Success("config written: %s", configPath)
	return nil
}

func deriveName(repo, _ string) string {
	parts := strings.Split(repo, "/")
	base := parts[len(parts)-1]
	base = strings.ToLower(base)
	base = strings.TrimSuffix(base, "-gguf")
	base = strings.TrimSuffix(base, "_gguf")
	base = strings.ReplaceAll(base, "_", "-")
	for _, prefix := range []string{"thebloke-", "bartowski-", "google-", "lmstudio-community-"} {
		base = strings.TrimPrefix(base, prefix)
	}
	return base
}
