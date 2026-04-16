package report

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/hiddenapi"
)

func TestDumpScanSARIF_ValidJSON(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleJava}
	DumpScanSARIF(&buf, result, dexFiles, "requestLocationUpdates", finder.ScopeAll, dc)

	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}
	if log.Version != "2.1.0" {
		t.Errorf("version = %q, want 2.1.0", log.Version)
	}
	if len(log.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(log.Runs))
	}
	run := log.Runs[0]
	if run.Tool.Driver.Name != "dexfinder" {
		t.Errorf("tool name = %q", run.Tool.Driver.Name)
	}
	if len(run.Tool.Driver.Rules) != 3 {
		t.Errorf("rules = %d, want 3", len(run.Tool.Driver.Rules))
	}
	if len(run.Results) == 0 {
		t.Error("should have results for requestLocationUpdates")
	}
	// Verify result structure
	for i, r := range run.Results {
		if r.RuleID == "" {
			t.Errorf("result %d: empty ruleId", i)
		}
		if r.Message.Text == "" {
			t.Errorf("result %d: empty message", i)
		}
		if r.Level == "" {
			t.Errorf("result %d: empty level", i)
		}
	}
}

func TestDumpTraceSARIF_ValidJSON(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleJava}
	DumpTraceSARIF(&buf, result, dexFiles, "requestLocationUpdates", 3, dc)

	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}
	if len(log.Runs) != 1 {
		t.Fatal("should have 1 run")
	}
	if len(log.Runs[0].Results) == 0 {
		t.Error("should have trace results")
	}
	for _, r := range log.Runs[0].Results {
		if r.RuleID != "dexfinder/call-chain" {
			t.Errorf("trace result ruleId = %q, want dexfinder/call-chain", r.RuleID)
		}
	}
}

func TestDumpHiddenAPISARIF_ValidJSON(t *testing.T) {
	csvPath := "../../testdata/hiddenapi-flags.csv"
	if _, err := os.Stat(csvPath); err != nil {
		t.Skip("hiddenapi-flags.csv not found")
	}
	dexFiles, result := loadHTMLTestFixtures(t)

	filter := hiddenapi.NewApiListFilter(nil)
	db := hiddenapi.NewDatabase(filter)
	db.LoadFromFile(csvPath)

	filtered := result.FilterHiddenAPIs(db)
	var buf bytes.Buffer
	DumpHiddenAPISARIF(&buf, filtered, dexFiles, db)

	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}
	if len(log.Runs) != 1 {
		t.Fatal("should have 1 run")
	}
	// Should have at least some hidden API results
	if len(log.Runs[0].Results) == 0 {
		t.Error("should have hidden API results")
	}
	// Check levels
	for _, r := range log.Runs[0].Results {
		switch r.Level {
		case "error", "warning", "note":
			// valid
		default:
			t.Errorf("invalid SARIF level: %q", r.Level)
		}
	}
}

func TestDumpScanSARIF_EmptyQuery(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleJava}
	// Empty query returns all results — just verify it doesn't panic and produces valid JSON
	DumpScanSARIF(&buf, result, dexFiles, "", finder.ScopeCallee, dc)

	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestHiddenAPILevel(t *testing.T) {
	tests := []struct {
		in   hiddenapi.ApiList
		want string
	}{
		{hiddenapi.Blocked, "error"},
		{hiddenapi.Unsupported, "warning"},
		{hiddenapi.MaxTargetO, "note"},
		{hiddenapi.Sdk, "note"},
	}
	for _, tt := range tests {
		if got := hiddenAPILevel(tt.in); got != tt.want {
			t.Errorf("hiddenAPILevel(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
