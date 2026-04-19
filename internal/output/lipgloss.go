package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	colorRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	colorYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	colorCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	colorGray   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold        = lipgloss.NewStyle().Bold(true)

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// RenderTable renders a table with Lip Gloss styling.
func RenderTable(headers []string, rows [][]string) string {
	if len(rows) == 0 && len(headers) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	// Header
	for i, h := range headers {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(headerStyle.Render(padRight(strings.ToUpper(h), widths[i])))
	}
	sb.WriteString("\n")

	// Separator
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(borderStyle.Render(strings.Repeat("─", w)))
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if i > 0 {
				sb.WriteString("  ")
			}
			sb.WriteString(padRight(cell, widths[i]))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func StatusColor(status string) string {
	switch status {
	case "running":
		return colorGreen.Render(status)
	case "stopped":
		return colorGray.Render(status)
	case "error":
		return colorRed.Render(status)
	default:
		return status
	}
}

func SuccessMsg(format string, args ...any) string {
	return colorGreen.Render("✓ " + fmt.Sprintf(format, args...))
}

func ErrorMsg(format string, args ...any) string {
	return colorRed.Render("✗ " + fmt.Sprintf(format, args...))
}

func InfoMsg(format string, args ...any) string {
	return colorCyan.Render("→ " + fmt.Sprintf(format, args...))
}

func WarnMsg(format string, args ...any) string {
	return colorYellow.Render("⚠ " + fmt.Sprintf(format, args...))
}

func Bold(s string) string {
	return bold.Render(s)
}
