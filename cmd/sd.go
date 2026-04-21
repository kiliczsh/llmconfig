package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
	"github.com/kiliczsh/llamaconfig/pkg/stablediffusioncpp"
	"github.com/spf13/cobra"
)

func newSdCmd() *cobra.Command {
	var flagPath bool
	var flagVersion bool
	var flagInstall bool
	var flagUpdate bool
	var flagBackend string

	cmd := &cobra.Command{
		Use:   "sd",
		Short: "Manage the stable-diffusion.cpp binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			if flagPath {
				path, err := stablediffusioncpp.FindBinary()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}

			if flagVersion {
				path, err := stablediffusioncpp.FindBinary()
				if err != nil {
					return err
				}
				ver, err := stablediffusioncpp.Version(path)
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
					case hardware.ClassAppleSilicon:
						backend = "metal"
					default:
						backend = "cpu"
					}
				}

				p.Info("fetching latest stable-diffusion.cpp release info...")
				rel, err := stablediffusioncpp.LatestRelease()
				if err != nil {
					return err
				}

				asset, err := stablediffusioncpp.PickAsset(rel, backend)
				if err != nil {
					return err
				}

				p.Info("release: %s", rel.TagName)
				p.Info("asset:   %s (%s)", asset.Name, humanize.Bytes(uint64(asset.Size)))
				p.Info("dest:    %s", stablediffusioncpp.BinDir())

				return runSdInstallWithProgress(asset, p)
			}

			// Default: show status
			path, err := stablediffusioncpp.FindBinary()
			if err != nil {
				p.Warn("sd not found — run: llamaconfig sd --install")
				return nil
			}
			ver, err := stablediffusioncpp.Version(path)
			if err != nil {
				ver = "(version unknown)"
			}
			fmt.Printf("  %-12s %s\n", "path:", path)
			fmt.Printf("  %-12s %s\n", "version:", ver)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagPath, "path", false, "print the binary path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "print the stable-diffusion.cpp version")
	cmd.Flags().BoolVar(&flagInstall, "install", false, "download and install the latest stable-diffusion.cpp binary")
	cmd.Flags().BoolVar(&flagUpdate, "update", false, "update stable-diffusion.cpp to the latest version")
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cuda | metal | cpu (default: auto-detect)")
	return cmd
}

// ── Install progress TUI ──────────────────────────────────────────────────────

type sdInstallDoneMsg struct{ err error }
type sdInstallProgressMsg struct{ downloaded, total int64 }

type sdInstallProgressModel struct {
	label      string
	downloaded int64
	total      int64
	done       bool
	err        error
}

func (m sdInstallProgressModel) Init() tea.Cmd { return nil }

func (m sdInstallProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sdInstallProgressMsg:
		m.downloaded = msg.downloaded
		m.total = msg.total
		return m, nil
	case sdInstallDoneMsg:
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

func (m sdInstallProgressModel) View() string {
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

func runSdInstallWithProgress(asset *stablediffusioncpp.GithubAsset, p interface{ Success(string, ...any) }) error {
	prog := tea.NewProgram(sdInstallProgressModel{label: asset.Name})

	go func() {
		err := stablediffusioncpp.Install(asset, func(downloaded, total int64) {
			prog.Send(sdInstallProgressMsg{downloaded: downloaded, total: total})
		})
		prog.Send(sdInstallDoneMsg{err: err})
	}()

	finalModel, err := prog.Run()
	if err != nil {
		return err
	}
	m := finalModel.(sdInstallProgressModel)
	if m.err != nil {
		return m.err
	}

	p.Success("stable-diffusion.cpp installed to %s", stablediffusioncpp.BinDir())
	p.Success("run: llamaconfig sd --version")
	return nil
}
