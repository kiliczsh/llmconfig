package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/kiliczsh/llamaconfig/internal/hardware"
	"github.com/kiliczsh/llamaconfig/pkg/llamacpp"
	"github.com/kiliczsh/llamaconfig/pkg/stablediffusioncpp"
	"github.com/kiliczsh/llamaconfig/pkg/whispercpp"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install AI inference binaries (llama, sd, whisper)",
	}
	cmd.AddCommand(
		newInstallLlamaCmd(),
		newInstallSdCmd(),
		newInstallWhisperCmd(),
	)
	return cmd
}

func newInstallLlamaCmd() *cobra.Command {
	var flagBackend string
	var flagFile string
	var flagVersion string

	cmd := &cobra.Command{
		Use:   "llama",
		Short: "Install llama.cpp (llama-server, llama-cli)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagFile != "" {
				p.Info("extracting %s...", flagFile)
				p.Info("dest:       %s", llamacpp.BinDir())
				if err := llamacpp.Extract(flagFile); err != nil {
					return err
				}
				p.Success("llama.cpp installed to %s", llamacpp.BinDir())
				p.Success("run: llamaconfig llama --version")
				return nil
			}

			backend := resolvedBackend(flagBackend, true)
			p.Info("fetching llama.cpp release info...")
			var (
				rel *llamacpp.GithubRelease
				err error
			)
			if flagVersion != "" {
				rel, err = llamacpp.ReleaseByTag(flagVersion)
			} else {
				rel, err = llamacpp.LatestRelease()
			}
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

			return runBinInstallWithProgress(asset.Name, asset.Size, func(onProgress func(int64, int64)) error {
				return llamacpp.Install(asset, onProgress)
			}, func() {
				p.Info("checksum verification: skipped (no published digest)")
				p.Success("llama.cpp installed to %s", llamacpp.BinDir())
				p.Success("run: llamaconfig llama --version")
			})
		},
	}
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cuda | metal | cpu (default: auto-detect)")
	cmd.Flags().StringVar(&flagFile, "file", "", "install from local zip instead of downloading")
	cmd.Flags().StringVar(&flagVersion, "version", "", "install a specific release tag (default: latest)")
	return cmd
}

func newInstallSdCmd() *cobra.Command {
	var flagBackend string
	var flagFile string
	var flagVersion string

	cmd := &cobra.Command{
		Use:   "sd",
		Short: "Install stable-diffusion.cpp (sd-cli, sd-server)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagFile != "" {
				p.Info("extracting %s...", flagFile)
				p.Info("dest:       %s", stablediffusioncpp.BinDir())
				if err := stablediffusioncpp.Extract(flagFile); err != nil {
					return err
				}
				p.Success("stable-diffusion.cpp installed to %s", stablediffusioncpp.BinDir())
				p.Success("run: llamaconfig sd --version")
				return nil
			}

			backend := resolvedBackend(flagBackend, false)
			p.Info("fetching stable-diffusion.cpp release info...")
			var (
				rel *stablediffusioncpp.GithubRelease
				err error
			)
			if flagVersion != "" {
				rel, err = stablediffusioncpp.ReleaseByTag(flagVersion)
			} else {
				rel, err = stablediffusioncpp.LatestRelease()
			}
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

			return runBinInstallWithProgress(asset.Name, asset.Size, func(onProgress func(int64, int64)) error {
				return stablediffusioncpp.Install(asset, onProgress)
			}, func() {
				p.Info("checksum verification: skipped (no published digest)")
				p.Success("stable-diffusion.cpp installed to %s", stablediffusioncpp.BinDir())
				p.Success("run: llamaconfig sd --version")
			})
		},
	}
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cuda | metal | cpu (default: auto-detect)")
	cmd.Flags().StringVar(&flagFile, "file", "", "install from local zip instead of downloading")
	cmd.Flags().StringVar(&flagVersion, "version", "", "install a specific release tag (default: latest)")
	return cmd
}

func newInstallWhisperCmd() *cobra.Command {
	var flagBackend string
	var flagFile string
	var flagVersion string

	cmd := &cobra.Command{
		Use:   "whisper",
		Short: "Install whisper.cpp (whisper-cli, whisper-server)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagFile != "" {
				p.Info("extracting %s...", flagFile)
				p.Info("dest:       %s", whispercpp.BinDir())
				if err := whispercpp.Extract(flagFile); err != nil {
					return err
				}
				p.Success("whisper.cpp installed to %s", whispercpp.BinDir())
				p.Success("run: llamaconfig whisper --version")
				return nil
			}

			backend := resolvedBackend(flagBackend, false)
			p.Info("fetching whisper.cpp release info...")
			var (
				rel *whispercpp.GithubRelease
				err error
			)
			if flagVersion != "" {
				rel, err = whispercpp.ReleaseByTag(flagVersion)
			} else {
				rel, err = whispercpp.LatestRelease()
			}
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

			return runBinInstallWithProgress(asset.Name, asset.Size, func(onProgress func(int64, int64)) error {
				return whispercpp.Install(asset, onProgress)
			}, func() {
				p.Info("checksum verification: skipped (no published digest)")
				p.Success("whisper.cpp installed to %s", whispercpp.BinDir())
				p.Success("run: llamaconfig whisper --version")
			})
		},
	}
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cuda | cpu (default: auto-detect)")
	cmd.Flags().StringVar(&flagFile, "file", "", "install from local zip instead of downloading")
	cmd.Flags().StringVar(&flagVersion, "version", "", "install a specific release tag (default: latest)")
	return cmd
}

// resolvedBackend auto-detects backend when flagBackend is empty.
// withMetal: llama supports Metal on Apple Silicon, sd/whisper do not.
func resolvedBackend(flagBackend string, withMetal bool) string {
	if flagBackend != "" {
		return flagBackend
	}
	hw := hardware.Detect()
	switch hw.Class {
	case hardware.ClassNVIDIA:
		return "cuda"
	case hardware.ClassAppleSilicon:
		if withMetal {
			return "metal"
		}
	}
	return "cpu"
}

// ── Binary install progress TUI ───────────────────────────────────────────────

type binDoneMsg struct{ err error }
type binProgressMsg struct{ downloaded, total int64 }

type binProgressModel struct {
	label      string
	downloaded int64
	total      int64
	done       bool
	err        error
}

func (m binProgressModel) Init() tea.Cmd { return nil }

func (m binProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case binProgressMsg:
		m.downloaded = msg.downloaded
		m.total = msg.total
	case binDoneMsg:
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

func (m binProgressModel) View() string {
	if m.done {
		if m.err != nil {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗ "+m.err.Error()) + "\n"
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓ installed: "+m.label) + "\n"
	}
	if m.total > 0 {
		pct := float64(m.downloaded) / float64(m.total) * 100
		return fmt.Sprintf("  downloading %s  %.0f%%  %s / %s\n",
			m.label, pct,
			humanize.Bytes(uint64(m.downloaded)),
			humanize.Bytes(uint64(m.total)))
	}
	return fmt.Sprintf("  downloading %s  %s\n", m.label, humanize.Bytes(uint64(m.downloaded)))
}

func runBinInstallWithProgress(label string, size int64, fn func(func(int64, int64)) error, onSuccess func()) error {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Printf("  downloading %s...\n", label)
		if err := fn(func(downloaded, total int64) {}); err != nil {
			return err
		}
		onSuccess()
		return nil
	}

	prog := tea.NewProgram(binProgressModel{label: label, total: size})
	go func() {
		err := fn(func(downloaded, total int64) {
			prog.Send(binProgressMsg{downloaded: downloaded, total: total})
		})
		prog.Send(binDoneMsg{err: err})
	}()
	finalModel, err := prog.Run()
	if err != nil {
		return err
	}
	m := finalModel.(binProgressModel)
	if m.err != nil {
		return m.err
	}
	onSuccess()
	return nil
}
