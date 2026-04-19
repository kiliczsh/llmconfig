package downloader

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type progressMsg struct {
	downloaded int64
	total      int64
}

type progressModel struct {
	bar        progress.Model
	downloaded int64
	total      int64
	label      string
	done       bool
	width      int
}

func newProgressModel(label string) progressModel {
	return progressModel{
		bar:   progress.New(progress.WithDefaultGradient()),
		label: label,
		width: 60,
	}
}

func (m progressModel) Init() tea.Cmd {
	return nil
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		m.downloaded = msg.downloaded
		m.total = msg.total
		if m.total > 0 && m.downloaded >= m.total {
			m.done = true
			return m, tea.Quit
		}
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
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m progressModel) View() string {
	if m.done {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(
			fmt.Sprintf("✓ %s (%s)", m.label, humanize.Bytes(uint64(m.downloaded))),
		) + "\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s\n", m.label))
	sb.WriteString(m.bar.View())
	if m.total > 0 {
		sb.WriteString(fmt.Sprintf("  %s / %s",
			humanize.Bytes(uint64(m.downloaded)),
			humanize.Bytes(uint64(m.total)),
		))
	} else {
		sb.WriteString(fmt.Sprintf("  %s", humanize.Bytes(uint64(m.downloaded))))
	}
	return sb.String()
}

// MakeProgressFunc returns a progress callback and a channel-based update sender for Bubble Tea.
func MakeProgressFunc(send func(tea.Msg)) func(downloaded, total int64) {
	return func(downloaded, total int64) {
		send(progressMsg{downloaded: downloaded, total: total})
	}
}
