package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
	"github.com/kiliczsh/llamaconfig/pkg/llamacpp"
	"github.com/spf13/cobra"
)

func newLlamaCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool
	var flagInstall bool
	var flagUpdate bool
	var flagBackend string

	cmd := &cobra.Command{
		Use:   "llama",
		Short: "Manage the llama.cpp binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			if flagPath {
				path, err := llamacpp.FindServer()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}

			if flagVersion {
				path, err := llamacpp.FindServer()
				if err != nil {
					return err
				}
				ver, err := llamacpp.Version(path)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n%s\n", path, ver)
				return nil
			}

			if flagInstall || flagUpdate {
				backend := flagBackend
				if backend == "" {
					// Auto-detect
					hw := hardware.Detect()
					switch hw.Class {
					case hardware.ClassNVIDIA:
						backend = "cuda"
					case hardware.ClassAppleSilicon:
						backend = "metal"
					default:
						backend = "cpu"
					}
				}

				p.Info("fetching latest llama.cpp release info...")
				rel, err := llamacpp.LatestRelease()
				if err != nil {
					return err
				}

				asset, err := llamacpp.PickAsset(rel, backend)
				if err != nil {
					return err
				}

				p.Info("release: %s", rel.TagName)
				p.Info("asset:   %s (%s)", asset.Name, humanize.Bytes(uint64(asset.Size)))
				p.Info("dest:    %s", llamacpp.BinDir())

				return runInstallWithProgress(asset, p)
			}

			// Default: show status
			path, err := llamacpp.FindServer()
			if err != nil {
				p.Warn("llama-server not found — run: llamaconfig llama --install")
				return nil
			}
			ver, err := llamacpp.Version(path)
			if err != nil {
				ver = "(version unknown)"
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the llama.cpp version")
	cmd.Flags().BoolVar(&flagInstall, "install", false, "download and install the latest llama.cpp binary")
	cmd.Flags().BoolVar(&flagUpdate, "update", false, "update llama.cpp to the latest version")
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cuda | metal | cpu (default: auto-detect)")
	return cmd
}

// ── Install progress TUI ──────────────────────────────────────────────────────

type installDoneMsg struct{ err error }
type installProgressMsg struct{ downloaded, total int64 }

type installProgressModel struct {
	label      string
	downloaded int64
	total      int64
	done       bool
	err        error
}

func (m installProgressModel) Init() tea.Cmd { return nil }

func (m installProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case installProgressMsg:
		m.downloaded = msg.downloaded
		m.total = msg.total
		return m, nil
	case installDoneMsg:
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

func (m installProgressModel) View() string {
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

func runInstallWithProgress(asset *llamacpp.GithubAsset, p interface{ Success(string, ...any) }) error {
	prog := tea.NewProgram(installProgressModel{label: asset.Name})

	go func() {
		err := llamacpp.Install(asset, func(downloaded, total int64) {
			prog.Send(installProgressMsg{downloaded: downloaded, total: total})
		})
		prog.Send(installDoneMsg{err: err})
	}()

	finalModel, err := prog.Run()
	if err != nil {
		return err
	}
	m := finalModel.(installProgressModel)
	if m.err != nil {
		return m.err
	}

	p.Success("llama.cpp installed to %s", llamacpp.BinDir())
	p.Success("run: llamaconfig llama --version")
	return nil
}
