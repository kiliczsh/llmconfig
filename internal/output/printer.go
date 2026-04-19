package output

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Printer is the output abstraction for all commands.
type Printer struct {
	NoColor bool
	JSON    bool
}

func New(noColor, jsonOutput bool) *Printer {
	return &Printer{NoColor: noColor, JSON: jsonOutput}
}

func (p *Printer) Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		p.Info("no entries")
		return
	}
	fmt.Print(RenderTable(headers, rows))
}

func (p *Printer) PrintJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (p *Printer) PrintYAML(v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
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
