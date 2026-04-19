package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kiliczsh/llamaconfig/internal/process"
	"github.com/kiliczsh/llamaconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var flagFollow bool
	var flagLines int
	var flagSince string

	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Show model logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			appCtx := appCtxFrom(cmd.Context())
			p := appCtx.Printer

			ms, err := appCtx.StateStore.Get(name)
			if err != nil {
				return err
			}
			if ms == nil {
				return fmt.Errorf("model %q not found — has it been started before?", name)
			}

			logFile := ms.LogFile
			if _, err := os.Stat(logFile); os.IsNotExist(err) {
				return fmt.Errorf("log file not found: %s", logFile)
			}

			_ = flagSince // TODO: filter by time

			if flagFollow {
				r := runner.New()
				return runLogsFollow(logFile, ms.Name, flagLines, r.IsAlive(ms))
			}

			lines, err := process.TailLines(logFile, flagLines)
			if err != nil {
				return err
			}

			for _, line := range lines {
				p.Info("%s", line)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagFollow, "follow", "f", false, "stream logs in real time")
	cmd.Flags().IntVarP(&flagLines, "lines", "n", 50, "number of lines to show")
	cmd.Flags().StringVar(&flagSince, "since", "", "show logs since duration (e.g. 1h, 30m)")
	return cmd
}

// ── Follow TUI ────────────────────────────────────────────────────────────────

type logLineMsg string
type logTickMsg struct{}

type logsModel struct {
	lines    []string
	name     string
	logFile  string
	stopCh   chan struct{}
	lineCh   chan string
	maxLines int
}

func newLogsModel(logFile, name string, initialLines []string) logsModel {
	m := logsModel{
		lines:    initialLines,
		name:     name,
		logFile:  logFile,
		stopCh:   make(chan struct{}),
		lineCh:   make(chan string, 100),
		maxLines: 200,
	}
	return m
}

func (m logsModel) Init() tea.Cmd {
	go func() {
		_ = process.Follow(m.logFile, m.lineCh, m.stopCh)
	}()
	return tickLogs()
}

func tickLogs() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return logTickMsg{}
	})
}

func (m logsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case logTickMsg:
		// Drain channel
		drained := false
		for !drained {
			select {
			case line := <-m.lineCh:
				m.lines = append(m.lines, line)
				if len(m.lines) > m.maxLines {
					m.lines = m.lines[len(m.lines)-m.maxLines:]
				}
			default:
				drained = true
			}
		}
		return m, tickLogs()

	case tea.KeyMsg:
		close(m.stopCh)
		return m, tea.Quit
	}
	return m, nil
}

func (m logsModel) View() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).
		Render(fmt.Sprintf("logs: %s  (q to quit)", m.name))

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	var body string
	for _, line := range m.lines {
		body += dim.Render("│ ") + line + "\n"
	}

	return header + "\n" + body
}

func runLogsFollow(logFile, name string, n int, _ bool) error {
	initial, err := process.TailLines(logFile, n)
	if err != nil {
		return err
	}
	m := newLogsModel(logFile, name, initial)
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// signature mismatch fix — match what's called above
func init() {
	_ = fmt.Sprintf
}
