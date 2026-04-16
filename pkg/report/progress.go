package report

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// Progress reports scanning progress to stderr.
type Progress struct {
	w       io.Writer
	enabled bool
	start   time.Time
}

// NewProgress creates a progress reporter.
// If w is nil, progress reporting is disabled.
func NewProgress(w io.Writer) *Progress {
	if w == nil {
		return &Progress{enabled: false}
	}
	return &Progress{w: w, enabled: true, start: time.Now()}
}

// Phase prints a phase header with a label.
func (p *Progress) Phase(label string) {
	if !p.enabled {
		return
	}
	fmt.Fprintf(p.w, "%s ...\n", label)
}

// Phasef prints a formatted phase header.
func (p *Progress) Phasef(format string, args ...interface{}) {
	if !p.enabled {
		return
	}
	fmt.Fprintf(p.w, format+" ...\n", args...)
}

// Detail prints a detail line under a phase.
func (p *Progress) Detail(format string, args ...interface{}) {
	if !p.enabled {
		return
	}
	fmt.Fprintf(p.w, format+"\n", args...)
}

// Done prints a completion message with elapsed time.
func (p *Progress) Done() {
	if !p.enabled {
		return
	}
	fmt.Fprintf(p.w, "Done in %v\n", time.Since(p.start))
}

// Elapsed returns duration since progress was created.
func (p *Progress) Elapsed() time.Duration {
	return time.Since(p.start)
}

// Counter is an atomic counter for tracking scan progress.
type Counter struct {
	val atomic.Int64
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.val.Add(1)
}

// Add increments the counter by n.
func (c *Counter) Add(n int64) {
	c.val.Add(n)
}

// Value returns the current value.
func (c *Counter) Value() int64 {
	return c.val.Load()
}
