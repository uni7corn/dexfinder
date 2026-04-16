package report

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestColorizer_NilSafe(t *testing.T) {
	var c *Colorizer
	if c.Enabled() {
		t.Error("nil colorizer should not be enabled")
	}
	// All methods should be nil-safe and return input unchanged
	if got := c.Tag("[METHOD]"); got != "[METHOD]" {
		t.Errorf("Tag = %q", got)
	}
	if got := c.TreeConnector("├──"); got != "├──" {
		t.Errorf("TreeConnector = %q", got)
	}
	if got := c.Highlight("hello world", "world"); got != "hello world" {
		t.Errorf("Highlight = %q", got)
	}
	if got := c.HiddenAPILevel("blocked"); got != "blocked" {
		t.Errorf("HiddenAPILevel = %q", got)
	}
}

func TestColorizer_Disabled(t *testing.T) {
	c := NewColorizer(ColorNever, os.Stdout)
	if c.Enabled() {
		t.Error("ColorNever should not be enabled")
	}
	if got := c.Tag("[METHOD]"); got != "[METHOD]" {
		t.Errorf("disabled Tag = %q", got)
	}
}

func TestColorizer_Enabled(t *testing.T) {
	c := NewColorizer(ColorAlways, os.Stdout)
	if !c.Enabled() {
		t.Error("ColorAlways should be enabled")
	}

	// Tag should contain ANSI codes
	tag := c.Tag("[METHOD]")
	if !strings.Contains(tag, "\033[") {
		t.Error("Tag should contain ANSI escape")
	}
	if !strings.Contains(tag, "[METHOD]") {
		t.Error("Tag should contain original text")
	}

	// Different tags get different colors
	fieldTag := c.Tag("[FIELD]")
	if tag == fieldTag {
		t.Error("METHOD and FIELD tags should have different colors")
	}

	// TreeConnector
	tc := c.TreeConnector("├── ")
	if !strings.Contains(tc, "\033[") {
		t.Error("TreeConnector should contain ANSI")
	}

	// Highlight
	hl := c.Highlight("com.foo.getDeviceId()", "DeviceId")
	if !strings.Contains(hl, "\033[") {
		t.Error("Highlight should contain ANSI")
	}
	if !strings.Contains(hl, "DeviceId") {
		t.Error("Highlight should preserve keyword")
	}

	// HiddenAPILevel
	blocked := c.HiddenAPILevel("blocked")
	unsupported := c.HiddenAPILevel("unsupported")
	if blocked == unsupported {
		t.Error("blocked and unsupported should have different colors")
	}

	// Cycle
	cycle := c.Cycle("⟳ [recursive]")
	if !strings.Contains(cycle, "\033[") {
		t.Error("Cycle should contain ANSI")
	}
}

func TestColorizer_AutoMode(t *testing.T) {
	// bytes.Buffer is not a terminal
	var buf bytes.Buffer
	c := NewColorizer(ColorAuto, &buf)
	if c.Enabled() {
		t.Error("auto mode with non-terminal should be disabled")
	}
}

func TestColorizer_HighlightNoMatch(t *testing.T) {
	c := NewColorizer(ColorAlways, os.Stdout)
	got := c.Highlight("hello world", "xyz")
	if strings.Contains(got, "\033[") {
		t.Error("no match should not add ANSI codes")
	}
}

func TestColorizer_HighlightEmpty(t *testing.T) {
	c := NewColorizer(ColorAlways, os.Stdout)
	got := c.Highlight("hello world", "")
	if got != "hello world" {
		t.Errorf("empty keyword should return input, got %q", got)
	}
}

func TestColorTag_AllTypes(t *testing.T) {
	c := NewColorizer(ColorAlways, os.Stdout)
	tags := []string{"[METHOD]", "[FIELD]", "[CALLER→]", "[STRING]", "[STRING_TABLE]"}
	for _, tag := range tags {
		got := c.Tag(tag)
		if !strings.Contains(got, "\033[") {
			t.Errorf("Tag(%s) should contain ANSI", tag)
		}
	}
}

// Test the nil-safe helper functions used in text.go
func TestColorHelpers_NilSafe(t *testing.T) {
	if got := colorTag(nil, "[METHOD]"); got != "[METHOD]" {
		t.Errorf("colorTag nil = %q", got)
	}
	if got := colorCount(nil, 5, "ref"); got != "(5 ref)" {
		t.Errorf("colorCount nil = %q", got)
	}
	if got := colorTreeConnector(nil, "├──"); got != "├──" {
		t.Errorf("colorTreeConnector nil = %q", got)
	}
	if got := colorChainHeader(nil, "header"); got != "header" {
		t.Errorf("colorChainHeader nil = %q", got)
	}
	if got := colorHiddenLevel(nil, "blocked"); got != "blocked" {
		t.Errorf("colorHiddenLevel nil = %q", got)
	}
	if got := colorSummary(nil, "summary"); got != "summary" {
		t.Errorf("colorSummary nil = %q", got)
	}
}
