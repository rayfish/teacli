// Package output handles formatted output for CLI commands
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Format represents the output format
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Printer handles output formatting
type Printer struct {
	format Format
	dryRun bool
}

// NewPrinter creates a new printer with the specified format
func NewPrinter(format Format) *Printer {
	return &Printer{
		format: format,
	}
}

// SetDryRun enables dry-run mode
func (p *Printer) SetDryRun(dryRun bool) {
	p.dryRun = dryRun
}

// SetFormat sets the output format
func (p *Printer) SetFormat(format Format) {
	p.format = format
}

// Format returns the current format
func (p *Printer) Format() Format {
	return p.format
}

// Print prints data in the configured format
func (p *Printer) Print(data interface{}) error {
	if p.dryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN: Would print %d items\n", p.countItems(data))
		return nil
	}

	switch p.format {
	case FormatJSON:
		return p.printJSON(data)
	default:
		return p.printText(data)
	}
}

// Emit prints data as JSON (the default) or, when the format is text, delegates to
// textFn for a human rendering. Errors and dry-run are handled centrally here.
func (p *Printer) Emit(data interface{}, textFn func() error) error {
	if p.dryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN: no changes made\n")
		return nil
	}
	if p.format == FormatText {
		if textFn == nil {
			return nil
		}
		return textFn()
	}
	return p.printJSON(data)
}

// PrintError prints an error in the configured format
func (p *Printer) PrintError(code int, message string, details map[string]interface{}) error {
	errorData := map[string]interface{}{
		"error":   true,
		"code":    code,
		"message": message,
	}
	if details != nil {
		errorData["details"] = details
	}

	data, _ := json.MarshalIndent(errorData, "", "  ")
	fmt.Fprintln(os.Stderr, string(data))
	return nil
}

// Println prints a simple line of text
func (p *Printer) Println(args ...interface{}) {
	fmt.Println(args...)
}

// Printf prints formatted text
func (p *Printer) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// PrintTable prints tabular data
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	if p.dryRun {
		fmt.Fprintf(os.Stderr, "DRY-RUN: Would print table with %d rows\n", len(rows))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print headers
	headerRow := make([]string, len(headers))
	for i, h := range headers {
		headerRow[i] = strings.ToUpper(h)
	}
	fmt.Fprintln(w, strings.Join(headerRow, "\t"))

	// Print separator
	separator := make([]string, len(headers))
	for i := range separator {
		separator[i] = strings.Repeat("-", len(headers[i]))
	}
	fmt.Fprintln(w, strings.Join(separator, "\t"))

	// Print rows
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	return w.Flush()
}

// printJSON prints data as JSON
func (p *Printer) printJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

// printText prints data in simple text format (one item per line) - DEPRECATED, kept for compatibility
func (p *Printer) printText(data interface{}) error {
	switch v := data.(type) {
	case []string:
		for _, item := range v {
			fmt.Println(item)
		}
	case []int:
		for _, item := range v {
			fmt.Println(item)
		}
	case []interface{}:
		for _, item := range v {
			fmt.Println(item)
		}
	default:
		fmt.Println(data)
	}
	return nil
}

// countItems returns the number of items in data (for dry-run messages)
func (p *Printer) countItems(data interface{}) int {
	switch v := data.(type) {
	case []interface{}:
		return len(v)
	case []string:
		return len(v)
	default:
		return 1
	}
}
