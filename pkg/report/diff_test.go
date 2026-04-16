package report

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"dex_method_finder/pkg/apk"
	"dex_method_finder/pkg/finder"
)

// --- Unit tests ---

func TestDumpDiffText_NoChanges(t *testing.T) {
	diff := &finder.DiffResult{}
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpDiffText(&buf, diff, dc)
	if !strings.Contains(buf.String(), "No differences") {
		t.Error("empty diff should say 'No differences'")
	}
}

func TestDumpDiffText_AddedMethods(t *testing.T) {
	diff := &finder.DiffResult{
		AddedMethods: []string{"Lcom/new/Foo;->bar()V", "Lcom/new/Baz;->qux()V"},
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpDiffText(&buf, diff, dc)
	out := buf.String()

	if !strings.Contains(out, "2 added method") {
		t.Error("should mention added count")
	}
	if !strings.Contains(out, "+") {
		t.Error("should contain + markers")
	}
	if !strings.Contains(out, "Lcom/new/Foo") {
		t.Error("should contain added API")
	}
}

func TestDumpDiffText_RemovedFields(t *testing.T) {
	diff := &finder.DiffResult{
		RemovedFields: []string{"Lcom/old/Foo;->field:I"},
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpDiffText(&buf, diff, dc)
	out := buf.String()

	if !strings.Contains(out, "1 removed field") {
		t.Error("should mention removed count")
	}
	if !strings.Contains(out, "-") {
		t.Error("should contain - markers")
	}
}

func TestDumpDiffText_Changed(t *testing.T) {
	diff := &finder.DiffResult{
		ChangedMethods: []finder.DiffEntry{
			{API: "Lcom/Foo;->bar()V", OldCount: 3, NewCount: 5},
		},
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpDiffText(&buf, diff, dc)
	out := buf.String()

	if !strings.Contains(out, "1 changed method") {
		t.Error("should mention changed count")
	}
	if !strings.Contains(out, "3 → 5") {
		t.Error("should show old → new counts")
	}
}

func TestDumpDiffText_AllCategories(t *testing.T) {
	diff := &finder.DiffResult{
		AddedMethods:   []string{"Lcom/new/A;->m()V"},
		AddedFields:    []string{"Lcom/new/A;->f:I"},
		RemovedMethods: []string{"Lcom/old/B;->m()V"},
		RemovedFields:  []string{"Lcom/old/B;->f:I"},
		ChangedMethods: []finder.DiffEntry{{API: "Lcom/C;->m()V", OldCount: 1, NewCount: 3}},
		ChangedFields:  []finder.DiffEntry{{API: "Lcom/C;->f:I", OldCount: 2, NewCount: 5}},
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpDiffText(&buf, diff, dc)
	out := buf.String()

	for _, expect := range []string{
		"1 added method", "1 added field",
		"1 removed method", "1 removed field",
		"1 changed method", "1 changed field",
		"+2 added", "-2 removed", "~2 changed",
	} {
		if !strings.Contains(out, expect) {
			t.Errorf("output should contain %q, got:\n%s", expect, out)
		}
	}
}

func TestDumpDiffText_Summary(t *testing.T) {
	diff := &finder.DiffResult{
		AddedMethods:   []string{"A"},
		RemovedMethods: []string{"B", "C"},
		ChangedFields:  []finder.DiffEntry{{API: "D", OldCount: 1, NewCount: 2}},
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpDiffText(&buf, diff, dc)
	out := buf.String()

	if !strings.Contains(out, "+1 added") {
		t.Errorf("summary should mention +1 added, got: %s", out)
	}
	if !strings.Contains(out, "-2 removed") {
		t.Errorf("summary should mention -2 removed, got: %s", out)
	}
	if !strings.Contains(out, "~1 changed") {
		t.Errorf("summary should mention ~1 changed, got: %s", out)
	}
}

func TestDumpDiffText_WithColors(t *testing.T) {
	diff := &finder.DiffResult{
		AddedMethods:   []string{"A"},
		RemovedMethods: []string{"B"},
		ChangedMethods: []finder.DiffEntry{{API: "C", OldCount: 1, NewCount: 3}},
	}
	var buf bytes.Buffer
	col := NewColorizer(ColorAlways, &buf)
	dc := &DisplayConfig{Color: col}
	DumpDiffText(&buf, diff, dc)
	out := buf.String()

	if !strings.Contains(out, "\033[") {
		t.Error("colored output should contain ANSI codes")
	}
	// Verify all sections are present despite coloring
	if !strings.Contains(out, "added") {
		t.Error("should contain 'added' section")
	}
	if !strings.Contains(out, "removed") {
		t.Error("should contain 'removed' section")
	}
}

// --- JSON tests ---

func TestDumpDiffJSON_Structure(t *testing.T) {
	diff := &finder.DiffResult{
		AddedMethods:   []string{"Lcom/new/Foo;->bar()V"},
		RemovedFields:  []string{"Lcom/old/Foo;->field:I"},
		ChangedMethods: []finder.DiffEntry{{API: "Lcom/Foo;->baz()V", OldCount: 2, NewCount: 5}},
	}
	var buf bytes.Buffer
	if err := DumpDiffJSON(&buf, diff); err != nil {
		t.Fatal(err)
	}

	// Verify JSON field names use snake_case (json tags)
	raw := buf.String()
	if !strings.Contains(raw, "added_methods") {
		t.Error("JSON should use snake_case key 'added_methods'")
	}
	if !strings.Contains(raw, "removed_fields") {
		t.Error("JSON should use snake_case key 'removed_fields'")
	}
	if !strings.Contains(raw, "changed_methods") {
		t.Error("JSON should use snake_case key 'changed_methods'")
	}
	if !strings.Contains(raw, "old_count") {
		t.Error("JSON should use snake_case key 'old_count'")
	}
	if !strings.Contains(raw, "new_count") {
		t.Error("JSON should use snake_case key 'new_count'")
	}

	// Verify round-trip
	var parsed finder.DiffResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed.AddedMethods) != 1 || parsed.AddedMethods[0] != "Lcom/new/Foo;->bar()V" {
		t.Errorf("AddedMethods = %v", parsed.AddedMethods)
	}
	if len(parsed.RemovedFields) != 1 {
		t.Errorf("RemovedFields = %d", len(parsed.RemovedFields))
	}
	if len(parsed.ChangedMethods) != 1 {
		t.Errorf("ChangedMethods = %d", len(parsed.ChangedMethods))
	}
	if parsed.ChangedMethods[0].OldCount != 2 || parsed.ChangedMethods[0].NewCount != 5 {
		t.Errorf("ChangedMethods[0] = %+v", parsed.ChangedMethods[0])
	}
}

func TestDumpDiffJSON_Empty(t *testing.T) {
	diff := &finder.DiffResult{}
	var buf bytes.Buffer
	DumpDiffJSON(&buf, diff)

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// With omitempty, empty slices should not appear
	if _, ok := parsed["added_methods"]; ok {
		t.Error("empty added_methods should be omitted")
	}
}

// --- Integration: diff with real APK ---

func TestDumpDiffText_RealAPK_Mutated(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	_ = dexFiles

	mutated := mutateResultForReport(result)

	diff := finder.DiffScans(result, mutated, nil, nil, "", finder.ScopeAll)

	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleJava}
	DumpDiffText(&buf, diff, dc)
	out := buf.String()

	if !strings.Contains(out, "added") {
		t.Error("should have added section")
	}
	if !strings.Contains(out, "removed") {
		t.Error("should have removed section")
	}
	if !strings.Contains(out, "Summary") {
		t.Error("should have summary")
	}
	t.Logf("Diff output length: %d bytes", len(out))
}

func TestDumpDiffJSON_RealAPK_Mutated(t *testing.T) {
	_, result, _ := loadTestFixtures(t)

	mutated := mutateResultForReport(result)
	diff := finder.DiffScans(result, mutated, nil, nil, "", finder.ScopeAll)

	var buf bytes.Buffer
	DumpDiffJSON(&buf, diff)

	var parsed finder.DiffResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed.TotalAdded() == 0 {
		t.Error("should have added entries")
	}
	if parsed.TotalRemoved() == 0 {
		t.Error("should have removed entries")
	}
}

// --- CLI integration test ---

func TestDiffCLI_SameFile(t *testing.T) {
	apkPath := "../../testdata/test.apk"
	if _, err := os.Stat(apkPath); err != nil {
		t.Skip("test.apk not found")
	}

	// Load both "old" and "new" from the same file
	oldDex, _ := apk.LoadDexFiles(apkPath)
	newDex, _ := apk.LoadDexFiles(apkPath)

	oldFinder := finder.NewDirectFinder(oldDex, finder.NewClassFilter(nil), nil)
	newFinder := finder.NewDirectFinder(newDex, finder.NewClassFilter(nil), nil)
	oldResult := oldFinder.Scan()
	newResult := newFinder.Scan()

	diff := finder.DiffScans(oldResult, newResult, oldDex, newDex, "", finder.ScopeAll)

	if diff.HasChanges() {
		t.Errorf("loading same APK twice should produce no diff, got: +%d -%d ~%d",
			diff.TotalAdded(), diff.TotalRemoved(), diff.TotalChanged())
	}
}

func TestDiffCLI_QueryFiltered(t *testing.T) {
	apkPath := "../../testdata/test.apk"
	if _, err := os.Stat(apkPath); err != nil {
		t.Skip("test.apk not found")
	}

	dex1, _ := apk.LoadDexFiles(apkPath)
	dex2, _ := apk.LoadDexFiles(apkPath)

	f1 := finder.NewDirectFinder(dex1, finder.NewClassFilter(nil), nil)
	f2 := finder.NewDirectFinder(dex2, finder.NewClassFilter(nil), nil)

	diff := finder.DiffScans(f1.Scan(), f2.Scan(), dex1, dex2, "requestLocationUpdates", finder.ScopeCallee)

	if diff.HasChanges() {
		t.Error("same file query-filtered diff should have no changes")
	}

	// Verify text output
	var buf bytes.Buffer
	dc := &DisplayConfig{Style: StyleJava}
	DumpDiffText(&buf, diff, dc)
	if !strings.Contains(buf.String(), "No differences") {
		t.Error("text output should say 'No differences'")
	}

	// Verify JSON output
	buf.Reset()
	DumpDiffJSON(&buf, diff)
	var parsed finder.DiffResult
	json.Unmarshal(buf.Bytes(), &parsed)
	if parsed.HasChanges() {
		t.Error("JSON diff should have no changes")
	}
}

// mutateResultForReport creates a modified ScanResult for report-level tests
func mutateResultForReport(orig *finder.ScanResult) *finder.ScanResult {
	mutated := &finder.ScanResult{
		MethodRefs: make(map[string][]finder.MethodRef),
		FieldRefs:  make(map[string][]finder.FieldRef),
		StringRefs: make(map[string][]finder.StringRef),
		Classes:    orig.Classes,
		AllStrings: orig.AllStrings,
	}

	i := 0
	for api, refs := range orig.MethodRefs {
		if i%8 == 0 {
			i++
			continue // remove
		}
		if i%5 == 0 {
			mutated.MethodRefs[api] = append(refs, refs[0]) // change count
		} else {
			mutated.MethodRefs[api] = refs
		}
		i++
	}
	mutated.MethodRefs["Lcom/test/new/V2Feature;->init()V"] = []finder.MethodRef{
		{CallerDexIdx: 0, CallerMethod: 1},
	}

	i = 0
	for api, refs := range orig.FieldRefs {
		if i%6 == 0 {
			i++
			continue
		}
		mutated.FieldRefs[api] = refs
		i++
	}

	return mutated
}
