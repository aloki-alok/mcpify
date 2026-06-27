// Package ui holds mcpify's terminal styling: ANSI colors that switch off when
// output is not a terminal, plus the wordmark. Keeping it here means the
// preview commands print plainly into pipes and CI while interactive use stays
// legible, and the stdio MCP transport on stdout is never polluted with color.
package ui

import "os"

// Style carries whether ANSI styling is on for a given writer.
type Style struct{ on bool }

// For returns the Style appropriate for f: styling is off when f is not a
// terminal, when NO_COLOR is set, or for dumb terminals.
func For(f *os.File) Style {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return Style{}
	}
	info, err := f.Stat()
	if err != nil {
		return Style{}
	}
	return Style{on: info.Mode()&os.ModeCharDevice != 0}
}

func (s Style) wrap(code, str string) string {
	if !s.on || str == "" {
		return str
	}
	return "\x1b[" + code + "m" + str + "\x1b[0m"
}

func (s Style) Bold(str string) string   { return s.wrap("1", str) }
func (s Style) Dim(str string) string    { return s.wrap("2", str) }
func (s Style) Accent(str string) string { return s.wrap("38;5;37", str) }
func (s Style) Green(str string) string  { return s.wrap("32", str) }
func (s Style) Red(str string) string    { return s.wrap("31", str) }

const art = "" +
	"                       _ ____\n" +
	"   ____ ___  _________(_) __/_  __\n" +
	"  / __ `__ \\/ ___/ __ \\/ /_/ / / /\n" +
	" / / / / / / /__/ /_/ / __/ /_/ /\n" +
	"/_/ /_/ /_/\\___/ .___/_/  \\__, /\n" +
	"              /_/        /____/"

// Banner returns the mcpify wordmark, accent-colored when styling is on.
func (s Style) Banner() string { return s.Accent(art) }
