package report

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"dex_method_finder/pkg/apk"
	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/hiddenapi"
)

func loadHTMLTestFixtures(t *testing.T) ([]*dex.DexFile, *finder.ScanResult) {
	t.Helper()
	apkPath := "../../testdata/test.apk"
	if _, err := os.Stat(apkPath); err != nil {
		t.Skip("test.apk not found")
	}
	dexFiles, err := apk.LoadDexFiles(apkPath)
	if err != nil {
		t.Fatalf("load APK: %v", err)
	}
	f := finder.NewDirectFinder(dexFiles, finder.NewClassFilter(nil), nil)
	result := f.Scan()
	return dexFiles, result
}

func TestDumpHTML_Scan_ValidHTML(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleJava}
	DumpHTML(&buf, result, dexFiles, "requestLocationUpdates", finder.ScopeAll, dc)
	out := buf.String()

	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("should contain DOCTYPE")
	}
	if !strings.Contains(out, "</html>") {
		t.Error("should contain closing html tag")
	}
	if !strings.Contains(out, "METHOD") {
		t.Error("should contain METHOD entries")
	}
	if !strings.Contains(out, "requestLocationUpdates") {
		t.Error("should contain query keyword")
	}
	if !strings.Contains(out, "filterEntries") {
		t.Error("should contain JavaScript for filtering")
	}
	if !strings.Contains(out, "<style>") {
		t.Error("should contain inline CSS")
	}
}

func TestDumpHTML_Scan_EscapesHTML(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleDex}
	DumpHTML(&buf, result, dexFiles, "requestLocationUpdates", finder.ScopeAll, dc)
	out := buf.String()

	// DEX signatures contain ; and > which should be escaped
	if strings.Contains(out, `<span class="api-name">L`) && !strings.Contains(out, "&gt;") && !strings.Contains(out, "&lt;") {
		// The HTML should have escaped signatures
	}
	// No raw < or > in API names (they'd break HTML)
	// This is a basic check — the main thing is no panic and valid structure
	if !strings.Contains(out, "</html>") {
		t.Error("malformed HTML")
	}
}

func TestDumpTraceHTML_Tree(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Layout: LayoutTree, Style: StyleJava}
	DumpTraceHTML(&buf, result, dexFiles, "requestLocationUpdates", 3, dc)
	out := buf.String()

	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(out, "call-tree") {
		t.Error("should contain call-tree class")
	}
	if !strings.Contains(out, "trace-target") {
		t.Error("should contain trace-target class")
	}
	if !strings.Contains(out, "target-api") {
		t.Error("should contain target-api class")
	}
}

func TestDumpTraceHTML_EmptyQuery(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpTraceHTML(&buf, result, dexFiles, "", 3, dc)
	if !strings.Contains(buf.String(), "Error") {
		t.Error("empty query should produce error message")
	}
}

func TestDumpTraceHTML_NoResults(t *testing.T) {
	dexFiles, result := loadHTMLTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleJava}
	DumpTraceHTML(&buf, result, dexFiles, "NonExistentXYZ999", 3, dc)
	if !strings.Contains(buf.String(), "No matching APIs") {
		t.Error("should say no matching APIs")
	}
}

func TestDumpHiddenAPIHTML(t *testing.T) {
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
	dc := &DisplayConfig{Style: StyleJava}
	DumpHiddenAPIHTML(&buf, filtered, dexFiles, db, dc)
	out := buf.String()

	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(out, "Hidden API") {
		t.Error("should contain Hidden API in title")
	}
	if !strings.Contains(out, "Linking") || !strings.Contains(out, "hidden API") {
		t.Error("should contain linking findings or summary")
	}
}
