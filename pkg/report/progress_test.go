package report

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestProgress_Basic(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)

	p.Phase("Loading")
	p.Detail("Loaded %d files", 5)
	p.Phasef("Scanning %s", "app.apk")
	time.Sleep(1 * time.Millisecond)
	p.Done()

	out := buf.String()

	if !strings.Contains(out, "Loading ...") {
		t.Error("should contain Phase output")
	}
	if !strings.Contains(out, "Loaded 5 files") {
		t.Error("should contain Detail output")
	}
	if !strings.Contains(out, "Scanning app.apk ...") {
		t.Error("should contain Phasef output")
	}
	if !strings.Contains(out, "Done in") {
		t.Error("should contain Done output")
	}
}

func TestProgress_Disabled(t *testing.T) {
	p := NewProgress(nil)

	// Should not panic
	p.Phase("Loading")
	p.Detail("detail %d", 1)
	p.Phasef("format %s", "test")
	p.Done()

	if p.Elapsed() < 0 {
		t.Error("elapsed should be non-negative")
	}
}

func TestProgress_Elapsed(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf)
	time.Sleep(5 * time.Millisecond)
	elapsed := p.Elapsed()
	if elapsed < 5*time.Millisecond {
		t.Errorf("elapsed too short: %v", elapsed)
	}
}

func TestCounter(t *testing.T) {
	c := &Counter{}
	if c.Value() != 0 {
		t.Error("initial value should be 0")
	}
	c.Inc()
	c.Inc()
	c.Add(3)
	if c.Value() != 5 {
		t.Errorf("value = %d, want 5", c.Value())
	}
}
