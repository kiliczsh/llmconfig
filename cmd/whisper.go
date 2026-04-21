package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
	"github.com/kiliczsh/llamaconfig/pkg/whispercpp"
	"github.com/spf13/cobra"
)

func newWhisperCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool
	var flagInstall bool
	var flagUpdate bool
	var flagBackend string

	cmd := &cobra.Command{
		Use:   "whisper",
		Short: "Manage the whisper.cpp binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			if flagPath {
				path, err := whispercpp.FindBinary()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}

			if flagVersion {
				path, err := whispercpp.FindBinary()
				if err != nil {
					return err
				}
				ver, err := whispercpp.Version(path)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n%s\n", path, ver)
				return nil
			}

			if flagInstall || flagUpdate {
				backend := flagBackend
				if backend == "" {
					hw := hardware.Detect()
					switch hw.Class {
					case hardware.ClassNVIDIA:
						backend = "cuda"
					default:
						backend = "cpu"
					}
				}

				p.Info("fetching latest whisper.cpp release info...")
				rel, err := whispercpp.LatestRelease()
				if err != nil {
					return err
				}

				asset, err := whispercpp.PickAsset(rel, backend)
				if err != nil {
					return err
				}

				p.Info("release: %s", rel.TagName)
				p.Info("asset:   %s (%s)", asset.Name, humanize.Bytes(uint64(asset.Size)))
				p.Info("dest:    %s", whispercpp.BinDir())

				return runWhisperInstallWithProgress(asset, p)
			}

			// Default: show status
			path, err := whispercpp.FindBinary()
			if err != nil {
				p.Warn("whisper-cli not found — run: llamaconfig whisper --install")
				return nil
			}
			ver, err := whispercpp.Version(path)
			if err != nil {
				ver = "(version unknown)"
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the whisper.cpp version")
	cmd.Flags().BoolVar(&flagInstall, "install", false, "download and install the latest whisper.cpp binary")
	cmd.Flags().BoolVar(&flagUpdate, "update", false, "update whisper.cpp to the latest version")
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cuda | cpu (default: auto-detect)")
	return cmd
}

// ── Install progress TUI ──────────────────────────────────────────────────────

type whisperInstallDoneMsg struct{ err error }
type whisperInstallProgressMsg struct{ downloaded, total int64 }

type whisperInstallProgressModel struct {
	label      string
	downloaded int64
	total      int64
	done       bool
	err        error
}

func (m whisperInstallProgressModel) Init() tea.Cmd { return nil }

func (m whisperInstallProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case whisperInstallProgressMsg:
		m.downloaded = msg.downloaded
		m.total = msg.total
		return m, nil
	case whisperInstallDoneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m whisperInstallProgressModel) View() string {
	if m.done {
		if m.err != nil {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗ "+m.err.Error()) + "\n"
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓ installed: "+m.label) + "\n"
	}
	if m.total > 0 {
		pct := float64(m.downloaded) / float64(m.total) * 100
		return fmt.Sprintf("Installing %s  %.0f%%  %s / %s\n",
			m.label, pct,
			humanize.Bytes(uint64(m.downloaded)),
			humanize.Bytes(uint64(m.total)))
	}
	return fmt.Sprintf("Installing %s  %s\n", m.label, humanize.Bytes(uint64(m.downloaded)))
}

func runWhisperInstallWithProgress(asset *whispercpp.GithubAsset, p interface{ Success(string, ...any) }) error {
	prog := tea.NewProgram(whisperInstallProgressModel{label: asset.Name})

	go func() {
		err := whispercpp.Install(asset, func(downloaded, total int64) {
			prog.Send(whisperInstallProgressMsg{downloaded: downloaded, total: total})
		})
		prog.Send(whisperInstallDoneMsg{err: err})
	}()

	finalModel, err := prog.Run()
	if err != nil {
		return err
	}
	m := finalModel.(whisperInstallProgressModel)
	if m.err != nil {
		return m.err
	}

	p.Success("whisper.cpp installed to %s", whispercpp.BinDir())
	p.Success("run: llamaconfig whisper --version")
	return nil
}
