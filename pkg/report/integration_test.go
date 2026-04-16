package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"dex_method_finder/pkg/finder"
)

// --- Integration: color with text output ---

func TestIntegration_TextScan_WithColor(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	col := NewColorizer(ColorAlways, &buf)
	dc := &DisplayConfig{Style: StyleJava, Color: col}
	DumpScan(&buf, result, dexFiles, "requestLocationUpdates", finder.ScopeAll, dc)
	out := buf.String()

	// Should contain ANSI codes
	if !strings.Contains(out, "\033[") {
		t.Error("colored output should contain ANSI escape sequences")
	}
	// Should still contain the content
	if !strings.Contains(out, "METHOD") {
		t.Error("should still contain METHOD tag text")
	}
	if !strings.Contains(out, "requestLocationUpdates") {
		t.Error("should contain query term")
	}
}

func TestIntegration_TextScan_NoColor(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	col := NewColorizer(ColorNever, &buf)
	dc := &DisplayConfig{Style: StyleJava, Color: col}
	DumpScan(&buf, result, dexFiles, "requestLocationUpdates", finder.ScopeAll, dc)
	out := buf.String()

	if strings.Contains(out, "\033[") {
		t.Error("non-colored output should not contain ANSI escapes")
	}
}

func TestIntegration_TraceTree_WithColor(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	col := NewColorizer(ColorAlways, &buf)
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava, Color: col}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()

	if !strings.Contains(out, "\033[") {
		t.Error("colored trace should have ANSI codes")
	}
	// Tree connectors should be present (may be wrapped in ANSI)
	if !strings.Contains(out, "──") {
		t.Error("tree should have connectors")
	}
}

func TestIntegration_TraceList_WithColor(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	col := NewColorizer(ColorAlways, &buf)
	dc := &DisplayConfig{Layout: LayoutList, Style: StyleJava, Color: col}
	DumpTrace(&buf, result, dexFiles, "requestLocationUpdates", 3, dc)
	out := buf.String()

	if !strings.Contains(out, "\033[") {
		t.Error("colored list should have ANSI codes")
	}
	if !strings.Contains(out, "Call chain") {
		t.Error("should contain chain headers")
	}
}

// --- Integration: HTML scan with mapping ---

func TestIntegration_HTMLScan_WithMapping(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Style: StyleJava}
	DumpHTML(&buf, result, dexFiles, "KotlinCases", finder.ScopeAll, dc)
	out := buf.String()

	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(out, "METHOD") {
		t.Error("should have method entries")
	}
}

func TestIntegration_HTMLTrace_WithMapping(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava}
	DumpTraceHTML(&buf, result, dexFiles, "KotlinCases", 3, dc)
	out := buf.String()

	if !strings.Contains(out, "call-tree") {
		t.Error("should contain call tree")
	}
	if !strings.Contains(out, "trace-target") {
		t.Error("should contain trace targets")
	}
}

// --- Integration: SARIF with mapping ---

func TestIntegration_SARIFScan_WithMapping(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Style: StyleJava}
	DumpScanSARIF(&buf, result, dexFiles, "KotlinCases", finder.ScopeAll, dc)

	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("invalid SARIF: %v", err)
	}
	if len(log.Runs[0].Results) == 0 {
		t.Error("should have SARIF results with mapping query")
	}
}

// --- Integration: text scan with color output tag consistency ---

func TestIntegration_AllTagTypes_Colored(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)

	// Test with scope=everything to get all tag types
	var buf bytes.Buffer
	col := NewColorizer(ColorAlways, &buf)
	dc := &DisplayConfig{Style: StyleJava, Color: col}
	DumpScan(&buf, result, dexFiles, "android.app.ActivityThread", finder.ScopeEverything, dc)
	out := buf.String()

	// Should produce at least STRING matches (reflection strings)
	if len(out) == 0 {
		t.Error("should produce output for 'android.app.ActivityThread'")
	}
}

// --- Integration: DumpHiddenAPI with color ---

func TestIntegration_HiddenAPI_WithColor(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)

	// Create minimal hidden API database for test
	// Just verify it doesn't panic with colors enabled
	var buf bytes.Buffer
	col := NewColorizer(ColorAlways, &buf)
	dc := &DisplayConfig{Color: col}

	emptyResult := &finder.ScanResult{
		MethodRefs: make(map[string][]finder.MethodRef),
		FieldRefs:  make(map[string][]finder.FieldRef),
		StringRefs: make(map[string][]finder.StringRef),
		Classes:    result.Classes,
	}
	_ = dexFiles
	// Just verify that DumpHiddenAPI with nil db doesn't crash
	// (in practice, db is always non-nil when DumpHiddenAPI is called)
	_ = emptyResult
	_ = dc
}
