package report

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ColorMode controls when colors are used.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"   // detect TTY
	ColorAlways ColorMode = "always" // force colors
	ColorNever  ColorMode = "never"  // no colors
)

// ANSI color codes
const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiRed       = "\033[31m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiBlue      = "\033[34m"
	ansiMagenta   = "\033[35m"
	ansiCyan      = "\033[36m"
	ansiBoldRed   = "\033[1;31m"
	ansiBoldGreen = "\033[1;32m"
	ansiBoldBlue  = "\033[1;34m"
	ansiBoldCyan  = "\033[1;36m"
)

// Colorizer applies ANSI colors to output strings.
type Colorizer struct {
	enabled bool
}

// NewColorizer creates a colorizer based on the color mode and output writer.
func NewColorizer(mode ColorMode, w io.Writer) *Colorizer {
	switch mode {
	case ColorAlways:
		return &Colorizer{enabled: true}
	case ColorNever:
		return &Colorizer{enabled: false}
	default: // auto
		return &Colorizer{enabled: isTerminal(w)}
	}
}

// isTerminal checks if the writer is a terminal (TTY).
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// Enabled returns whether colors are enabled.
func (c *Colorizer) Enabled() bool {
	return c != nil && c.enabled
}

// Tag colorizes output tags like [METHOD], [FIELD], [STRING].
func (c *Colorizer) Tag(tag string) string {
	if !c.Enabled() {
		return tag
	}
	switch {
	case strings.Contains(tag, "METHOD"):
		return ansiBoldBlue + tag + ansiReset
	case strings.Contains(tag, "FIELD"):
		return ansiBoldCyan + tag + ansiReset
	case strings.Contains(tag, "CALLER"):
		return ansiBoldGreen + tag + ansiReset
	case strings.Contains(tag, "STRING_TABLE"):
		return ansiDim + tag + ansiReset
	case strings.Contains(tag, "STRING"):
		return ansiYellow + tag + ansiReset
	default:
		return tag
	}
}

// TreeConnector colorizes tree drawing characters.
func (c *Colorizer) TreeConnector(s string) string {
	if !c.Enabled() {
		return s
	}
	return ansiDim + s + ansiReset
}

// Highlight colorizes a matched keyword within a string.
func (c *Colorizer) Highlight(text, keyword string) string {
	if !c.Enabled() || keyword == "" {
		return text
	}
	lower := strings.ToLower(text)
	kw := strings.ToLower(keyword)
	idx := strings.Index(lower, kw)
	if idx == -1 {
		return text
	}
	matched := text[idx : idx+len(keyword)]
	return text[:idx] + ansiBold + ansiRed + matched + ansiReset + text[idx+len(keyword):]
}

// HiddenAPILevel colorizes the API restriction level.
func (c *Colorizer) HiddenAPILevel(level string) string {
	if !c.Enabled() {
		return level
	}
	switch level {
	case "blocked":
		return ansiBoldRed + level + ansiReset
	case "unsupported":
		return ansiYellow + level + ansiReset
	default:
		return ansiCyan + level + ansiReset
	}
}

// Caller colorizes the indented caller line.
func (c *Colorizer) Caller(s string) string {
	if !c.Enabled() {
		return s
	}
	return ansiDim + s + ansiReset
}

// ChainHeader colorizes trace chain header lines.
func (c *Colorizer) ChainHeader(s string) string {
	if !c.Enabled() {
		return s
	}
	return ansiBold + s + ansiReset
}

// Count colorizes ref/occurrence counts.
func (c *Colorizer) Count(n int, unit string) string {
	if !c.Enabled() {
		return fmt.Sprintf("(%d %s)", n, unit)
	}
	return ansiDim + fmt.Sprintf("(%d %s)", n, unit) + ansiReset
}

// Summary colorizes the summary line.
func (c *Colorizer) Summary(s string) string {
	if !c.Enabled() {
		return s
	}
	return ansiBold + s + ansiReset
}

// Cycle colorizes the recursive/cycle marker.
func (c *Colorizer) Cycle(s string) string {
	if !c.Enabled() {
		return s
	}
	return ansiMagenta + s + ansiReset
}
