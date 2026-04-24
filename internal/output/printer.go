package output

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Printer is the output abstraction for all commands.
type Printer struct {
	NoColor bool
}

func New(noColor bool) *Printer {
	// Respect the de facto NO_COLOR standard (https://no-color.org).
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		noColor = true
	}
	if noColor {
		// Force lipgloss's package-global renderer into Ascii so *every*
		// style call (tables, StatusColor, Bold, the handful of direct
		// lipgloss calls in cmd/*) becomes a no-op. Without this, only
		// the Printer.Info/Success/… wrappers honored NoColor.
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	return &Printer{NoColor: noColor}
}

func (p *Printer) Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		p.Info("no entries")
		return
	}
	fmt.Print(RenderTable(headers, rows))
}

func (p *Printer) Success(format string, args ...any) {
	if p.NoColor {
		fmt.Printf("OK: "+format+"\n", args...)
	} else {
		fmt.Println(SuccessMsg(format, args...))
	}
}

func (p *Printer) Error(format string, args ...any) {
	if p.NoColor {
		fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	} else {
		fmt.Fprintln(os.Stderr, ErrorMsg(format, args...))
	}
}

func (p *Printer) Info(format string, args ...any) {
	if p.NoColor {
		fmt.Printf(format+"\n", args...)
	} else {
		fmt.Println(InfoMsg(format, args...))
	}
}

func (p *Printer) Warn(format string, args ...any) {
	if p.NoColor {
		fmt.Fprintf(os.Stderr, "WARN: "+format+"\n", args...)
	} else {
		fmt.Fprintln(os.Stderr, WarnMsg(format, args...))
	}
}
