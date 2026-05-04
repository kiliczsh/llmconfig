package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kiliczsh/llmconfig/internal/dirs"
	"github.com/kiliczsh/llmconfig/pkg/ikllamacpp"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newInstallIkLlamaCmd() *cobra.Command {
	var flagBackend string
	var flagRef string
	var flagFile string
	var flagVerbose bool
	var flagJobs int

	cmd := &cobra.Command{
		Use:   "ik_llama",
		Short: "Install ik_llama.cpp (build from source — git + cmake required)",
		Long: `Install ik_llama.cpp, ikawrakow's llama.cpp fork with SOTA quants and
better CPU/hybrid performance.

ik_llama.cpp ships no prebuilt release binaries, so this command compiles
from source: it clones (or updates) the repo into ~/.llmconfig/cache/
and runs cmake to produce llama-server / llama-cli, which are copied to
~/.llmconfig/bin/ik-llama/.

Prerequisites: git, cmake, a C++ compiler (and CUDA toolkit for --backend=cuda).
On Windows, run from a "Developer PowerShell for VS" so cl.exe is on PATH.

Use --file <archive.zip|.tar.gz> to skip the build entirely and install
from a prebuilt local archive (bring-your-own-binary).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := appCtxFrom(cmd.Context()).Printer

			if flagFile != "" {
				p.Info("extracting %s...", flagFile)
				p.Info("dest:       %s", ikllamacpp.BinDir())
				if err := ikllamacpp.Extract(flagFile); err != nil {
					return err
				}
				p.Success("ik_llama.cpp installed to %s", ikllamacpp.BinDir())
				p.Success("run: llmconfig ik_llama --version")
				return nil
			}

			backend := resolvedBackend(flagBackend, false)
			if backend == "metal" || backend == "rocm" || backend == "sycl" {
				p.Warn("ik_llama.cpp officially supports CPU and CUDA only — falling back to cpu")
				backend = "cpu"
			}

			logPath := filepath.Join(dirs.LogDir(), "install-ik-llama.log")
			p.Info("source:  %s", ikllamacpp.CacheDir())
			p.Info("dest:    %s", ikllamacpp.BinDir())
			p.Info("backend: %s", backend)
			if flagRef != "" {
				p.Info("ref:     %s", flagRef)
			}
			p.Info("log:     %s", logPath)

			opts := ikllamacpp.BuildOptions{
				Backend: backend,
				Ref:     flagRef,
				Jobs:    flagJobs,
				Verbose: flagVerbose,
				LogPath: logPath,
			}

			if flagVerbose || !isatty.IsTerminal(os.Stdin.Fd()) {
				// Non-TTY or explicit --verbose: stream cmake output directly,
				// no spinner. The build is long enough that a quiet terminal
				// looks broken; keep the user in the loop.
				opts.OnStep = func(msg string) {
					p.Info("→ %s", msg)
				}
				if err := ikllamacpp.Build(cmd.Context(), opts); err != nil {
					return fmt.Errorf("install ik_llama: %w (see %s)", err, logPath)
				}
				p.Success("ik_llama.cpp built and installed to %s", ikllamacpp.BinDir())
				p.Success("run: llmconfig ik_llama --version")
				return nil
			}

			return runIkLlamaBuildTUI(cmd.Context(), opts, logPath, p.Success)
		},
	}
	cmd.Flags().StringVar(&flagBackend, "backend", "", "backend: cpu | cuda (default: auto-detect)")
	cmd.Flags().StringVar(&flagRef, "ref", "", "git ref to build (tag, branch, or commit; default: main HEAD)")
	cmd.Flags().StringVar(&flagFile, "file", "", "install from local archive instead of building from source")
	cmd.Flags().BoolVar(&flagVerbose, "verbose", false, "stream build output to stderr in addition to the log file")
	cmd.Flags().IntVar(&flagJobs, "jobs", 0, fmt.Sprintf("parallel build jobs (default: %d)", runtime.NumCPU()))
	return cmd
}

// ── Build progress TUI ────────────────────────────────────────────────────────

type ikStepMsg string
type ikDoneMsg struct{ err error }
type ikTickMsg time.Time

type ikBuildModel struct {
	step    string
	elapsed time.Duration
	start   time.Time
	logPath string
	done    bool
	err     error
	spinner int
}

var ikSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m ikBuildModel) Init() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return ikTickMsg(t) })
}

func (m ikBuildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ikStepMsg:
		m.step = string(msg)
	case ikDoneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case ikTickMsg:
		m.elapsed = time.Since(m.start)
		m.spinner = (m.spinner + 1) % len(ikSpinnerFrames)
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return ikTickMsg(t) })
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ikBuildModel) View() string {
	if m.done {
		if m.err != nil {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗ "+m.err.Error()) + "\n"
		}
		return ""
	}
	step := m.step
	if step == "" {
		step = "starting…"
	}
	return fmt.Sprintf("  %s %s  (%s elapsed — tail %s for live output)\n",
		ikSpinnerFrames[m.spinner], step, m.elapsed.Truncate(time.Second), m.logPath)
}

func runIkLlamaBuildTUI(ctx context.Context, opts ikllamacpp.BuildOptions, logPath string, onSuccess func(string, ...any)) error {
	model := ikBuildModel{start: time.Now(), logPath: logPath}
	prog := tea.NewProgram(model)

	opts.OnStep = func(msg string) {
		prog.Send(ikStepMsg(msg))
	}

	go func() {
		err := ikllamacpp.Build(ctx, opts)
		prog.Send(ikDoneMsg{err: err})
	}()

	finalModel, err := prog.Run()
	if err != nil {
		return err
	}
	m := finalModel.(ikBuildModel)
	if m.err != nil {
		return fmt.Errorf("install ik_llama: %w (see %s)", m.err, logPath)
	}
	onSuccess("ik_llama.cpp built and installed to %s", ikllamacpp.BinDir())
	onSuccess("run: llmconfig ik_llama --version")
	return nil
}
