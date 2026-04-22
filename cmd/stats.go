package cmd

import (
	"fmt"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kiliczsh/llamaconfig/internal/output"
	"github.com/kiliczsh/llamaconfig/internal/process"
	"github.com/kiliczsh/llamaconfig/internal/runner"
	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	var flagWatch bool
	var flagInterval string

	cmd := &cobra.Command{
		Use:   "stats [name]",
		Short: "Show resource usage of running models",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())

			interval := 2 * time.Second
			if flagInterval != "" {
				if d, err := time.ParseDuration(flagInterval); err == nil {
					interval = d
				}
			}

			if flagWatch {
				return runStatsWatch(appCtx, interval)
			}

			return printStatsOnce(appCtx)
		},
	}

	cmd.Flags().BoolVarP(&flagWatch, "watch", "w", false, "live updating display")
	cmd.Flags().StringVar(&flagInterval, "interval", "2s", "watch refresh interval")
	return cmd
}

func printStatsOnce(appCtx *AppContext) error {
	p := appCtx.Printer
	rows, err := gatherStats(appCtx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		p.Info("no running models")
		return nil
	}
	headers := []string{"NAME", "PID", "PORT", "UPTIME", "CPU%", "MEM (MB)"}
	tableRows := statsToRows(rows)
	p.Table(headers, tableRows)
	return nil
}

type statsRow struct {
	Name   string
	PID    int
	Port   int
	Uptime string
	CPU    string
	MemMB  string
}

func gatherStats(appCtx *AppContext) ([]statsRow, error) {
	sf, err := appCtx.StateStore.Load()
	if err != nil {
		return nil, err
	}

	r := runner.New()
	names := make([]string, 0)
	for name := range sf.Models {
		names = append(names, name)
	}
	sort.Strings(names)

	var rows []statsRow
	for _, name := range names {
		ms := sf.Models[name]
		if ms.Status != "running" || !r.IsAlive(ms) {
			continue
		}

		row := statsRow{
			Name:   ms.Name,
			PID:    ms.PID,
			Port:   ms.Port,
			Uptime: output.FormatUptime(ms.StartedAt),
			CPU:    "-",
			MemMB:  "-",
		}

		if stats, err := process.GetStats(ms.PID); err == nil {
			row.CPU = fmt.Sprintf("%.1f", stats.CPUPercent)
			row.MemMB = fmt.Sprintf("%.0f", stats.MemoryMB)
		}

		rows = append(rows, row)
	}
	return rows, nil
}

func statsToRows(rows []statsRow) [][]string {
	result := make([][]string, len(rows))
	for i, r := range rows {
		result[i] = []string{
			r.Name,
			fmt.Sprintf("%d", r.PID),
			fmt.Sprintf("%d", r.Port),
			r.Uptime,
			r.CPU,
			r.MemMB,
		}
	}
	return result
}

// ── Watch TUI ─────────────────────────────────────────────────────────────────

type statsTickMsg time.Time

type statsWatchModel struct {
	appCtx   *AppContext
	rows     []statsRow
	interval time.Duration
	err      error
}

func (m statsWatchModel) Init() tea.Cmd {
	return statsTickCmd(m.interval)
}

func statsTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return statsTickMsg(t)
	})
}

func (m statsWatchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case statsTickMsg:
		rows, err := gatherStats(m.appCtx)
		m.rows = rows
		m.err = err
		return m, statsTickCmd(m.interval)

	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m statsWatchModel) View() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).
		Render("stats  (q to quit)")

	if m.err != nil {
		return header + "\n" + output.ErrorMsg("%s", m.err.Error())
	}

	if len(m.rows) == 0 {
		return header + "\n" + output.InfoMsg("no running models")
	}

	headers := []string{"NAME", "PID", "PORT", "UPTIME", "CPU%", "MEM (MB)"}
	return header + "\n" + output.RenderTable(headers, statsToRows(m.rows))
}

func runStatsWatch(appCtx *AppContext, interval time.Duration) error {
	rows, gErr := gatherStats(appCtx)
	m := statsWatchModel{appCtx: appCtx, rows: rows, err: gErr, interval: interval}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

