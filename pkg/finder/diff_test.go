package finder

import (
	"os"
	"sort"
	"testing"

	"dex_method_finder/pkg/apk"
)

// --- Helper ---

func loadRealAPK(t *testing.T) *ScanResult {
	t.Helper()
	apkPath := "../../testdata/test.apk"
	if _, err := os.Stat(apkPath); err != nil {
		t.Skip("test.apk not found")
	}
	dexFiles, err := apk.LoadDexFiles(apkPath)
	if err != nil {
		t.Fatal(err)
	}
	f := NewDirectFinder(dexFiles, NewClassFilter(nil), nil)
	return f.Scan()
}

// mutateResult creates a modified copy of ScanResult to simulate a different APK version.
// It removes some methods, adds some methods, and changes caller counts on some methods.
func mutateResult(orig *ScanResult) *ScanResult {
	mutated := &ScanResult{
		MethodRefs: make(map[string][]MethodRef),
		FieldRefs:  make(map[string][]FieldRef),
		StringRefs: make(map[string][]StringRef),
		Classes:    orig.Classes,
		AllStrings: orig.AllStrings,
	}

	// Copy all methods but skip every 10th (simulating removed APIs)
	i := 0
	removedCount := 0
	for api, refs := range orig.MethodRefs {
		if i%10 == 0 {
			removedCount++
			i++
			continue // skip this one (removed in new version)
		}
		if i%7 == 0 {
			// Add an extra ref to simulate changed caller count
			extra := refs[0]
			mutated.MethodRefs[api] = append(refs, extra)
		} else {
			mutated.MethodRefs[api] = refs
		}
		i++
	}

	// Add a synthetic new API (simulating added in new version)
	mutated.MethodRefs["Lcom/test/new/Feature;->newMethod()V"] = []MethodRef{
		{CallerDexIdx: 0, CallerMethod: 1},
	}
	mutated.MethodRefs["Lcom/test/new/Feature;->anotherNewMethod(I)Z"] = []MethodRef{
		{CallerDexIdx: 0, CallerMethod: 2},
		{CallerDexIdx: 0, CallerMethod: 3},
	}

	// Copy fields, remove every 5th
	i = 0
	for api, refs := range orig.FieldRefs {
		if i%5 == 0 {
			i++
			continue
		}
		mutated.FieldRefs[api] = refs
		i++
	}

	// Add a new field
	mutated.FieldRefs["Lcom/test/new/Feature;->newField:Ljava/lang/String;"] = []FieldRef{
		{CallerDexIdx: 0, CallerMethod: 1},
	}

	return mutated
}

// --- Tests ---

func TestDiffScans_SameFile_NoDifferences(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)

	diff := DiffScans(result, result, dexFiles, dexFiles, "", ScopeAll)
	if diff.HasChanges() {
		t.Errorf("same file diff should have no changes, got: added=%d removed=%d changed=%d",
			diff.TotalAdded(), diff.TotalRemoved(), diff.TotalChanged())
	}
}

func TestDiffScans_SameFile_WithQuery_NoDifferences(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)

	diff := DiffScans(result, result, dexFiles, dexFiles, "requestLocationUpdates", ScopeCallee)
	if diff.HasChanges() {
		t.Error("same file diff with query should have no changes")
	}
}

func TestDiffScans_MutatedAPK_DetectsAdded(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)
	mutated := mutateResult(result)

	// old=original, new=mutated → should see new APIs as added
	diff := DiffScans(result, mutated, dexFiles, dexFiles, "", ScopeAll)

	if !diff.HasChanges() {
		t.Fatal("mutated diff should have changes")
	}

	// The two synthetic methods should be in AddedMethods
	addedSet := make(map[string]bool)
	for _, a := range diff.AddedMethods {
		addedSet[a] = true
	}
	if !addedSet["Lcom/test/new/Feature;->newMethod()V"] {
		t.Error("should detect newMethod as added")
	}
	if !addedSet["Lcom/test/new/Feature;->anotherNewMethod(I)Z"] {
		t.Error("should detect anotherNewMethod as added")
	}

	// The synthetic field should be in AddedFields
	fieldAdded := false
	for _, a := range diff.AddedFields {
		if a == "Lcom/test/new/Feature;->newField:Ljava/lang/String;" {
			fieldAdded = true
		}
	}
	if !fieldAdded {
		t.Error("should detect newField as added")
	}

	t.Logf("Diff: +%d added, -%d removed, ~%d changed methods; +%d added, -%d removed fields",
		len(diff.AddedMethods), len(diff.RemovedMethods), len(diff.ChangedMethods),
		len(diff.AddedFields), len(diff.RemovedFields))
}

func TestDiffScans_MutatedAPK_DetectsRemoved(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)
	mutated := mutateResult(result)

	// old=original, new=mutated → every 10th method from old was removed in new
	diff := DiffScans(result, mutated, dexFiles, dexFiles, "", ScopeAll)

	if len(diff.RemovedMethods) == 0 {
		t.Error("should detect removed methods")
	}

	// Verify all removed methods actually existed in old
	for _, rm := range diff.RemovedMethods {
		if _, ok := result.MethodRefs[rm]; !ok {
			t.Errorf("removed method %s was not in old result", rm)
		}
		if _, ok := mutated.MethodRefs[rm]; ok {
			t.Errorf("removed method %s still in new result", rm)
		}
	}

	if len(diff.RemovedFields) == 0 {
		t.Error("should detect removed fields")
	}

	t.Logf("Removed: %d methods, %d fields", len(diff.RemovedMethods), len(diff.RemovedFields))
}

func TestDiffScans_MutatedAPK_DetectsChanged(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)
	mutated := mutateResult(result)

	diff := DiffScans(result, mutated, dexFiles, dexFiles, "", ScopeAll)

	if len(diff.ChangedMethods) == 0 {
		t.Error("should detect changed methods (different caller counts)")
	}

	// Verify changed entries have correct counts
	for _, ch := range diff.ChangedMethods {
		oldRefs := result.MethodRefs[ch.API]
		newRefs := mutated.MethodRefs[ch.API]
		if ch.OldCount != len(oldRefs) {
			t.Errorf("changed %s: OldCount=%d, actual old refs=%d", ch.API, ch.OldCount, len(oldRefs))
		}
		if ch.NewCount != len(newRefs) {
			t.Errorf("changed %s: NewCount=%d, actual new refs=%d", ch.API, ch.NewCount, len(newRefs))
		}
		if ch.OldCount == ch.NewCount {
			t.Errorf("changed %s should have different counts, both are %d", ch.API, ch.OldCount)
		}
	}

	t.Logf("Changed: %d methods", len(diff.ChangedMethods))
}

func TestDiffScans_MutatedAPK_WithQuery(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)
	mutated := mutateResult(result)

	// Query-filtered diff: only compare requestLocationUpdates APIs
	diff := DiffScans(result, mutated, dexFiles, dexFiles, "requestLocationUpdates", ScopeCallee)

	// requestLocationUpdates has 3 overloads in test.apk
	// Some may be removed (every 10th) or changed (every 7th)
	// The exact result depends on iteration order, but it should be consistent
	t.Logf("Query-filtered diff: +%d -%d ~%d",
		len(diff.AddedMethods), len(diff.RemovedMethods), len(diff.ChangedMethods))
}

func TestDiffScans_EmptyOld_AllAdded(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)

	emptyResult := &ScanResult{
		MethodRefs: make(map[string][]MethodRef),
		FieldRefs:  make(map[string][]FieldRef),
		StringRefs: make(map[string][]StringRef),
		Classes:    make(map[string]bool),
		AllStrings: make(map[string]bool),
	}

	diff := DiffScans(emptyResult, result, nil, dexFiles, "", ScopeAll)

	// All APIs in result should be "added"
	if diff.TotalAdded() != len(result.MethodRefs)+len(result.FieldRefs) {
		t.Errorf("TotalAdded=%d, want %d (all methods+fields)",
			diff.TotalAdded(), len(result.MethodRefs)+len(result.FieldRefs))
	}
	if diff.TotalRemoved() != 0 {
		t.Errorf("TotalRemoved=%d, want 0", diff.TotalRemoved())
	}
	if diff.TotalChanged() != 0 {
		t.Errorf("TotalChanged=%d, want 0", diff.TotalChanged())
	}
}

func TestDiffScans_EmptyNew_AllRemoved(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)

	emptyResult := &ScanResult{
		MethodRefs: make(map[string][]MethodRef),
		FieldRefs:  make(map[string][]FieldRef),
		StringRefs: make(map[string][]StringRef),
		Classes:    make(map[string]bool),
		AllStrings: make(map[string]bool),
	}

	diff := DiffScans(result, emptyResult, dexFiles, nil, "", ScopeAll)

	if diff.TotalRemoved() != len(result.MethodRefs)+len(result.FieldRefs) {
		t.Errorf("TotalRemoved=%d, want %d (all methods+fields)",
			diff.TotalRemoved(), len(result.MethodRefs)+len(result.FieldRefs))
	}
	if diff.TotalAdded() != 0 {
		t.Errorf("TotalAdded=%d, want 0", diff.TotalAdded())
	}
}

func TestDiffScans_Deterministic(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)
	mutated := mutateResult(result)

	// Run diff twice, should produce same results
	diff1 := DiffScans(result, mutated, dexFiles, dexFiles, "", ScopeAll)
	diff2 := DiffScans(result, mutated, dexFiles, dexFiles, "", ScopeAll)

	if len(diff1.AddedMethods) != len(diff2.AddedMethods) {
		t.Error("non-deterministic AddedMethods count")
	}
	if len(diff1.RemovedMethods) != len(diff2.RemovedMethods) {
		t.Error("non-deterministic RemovedMethods count")
	}
	if len(diff1.ChangedMethods) != len(diff2.ChangedMethods) {
		t.Error("non-deterministic ChangedMethods count")
	}

	// Verify sorted output
	if !sort.StringsAreSorted(diff1.AddedMethods) {
		t.Error("AddedMethods should be sorted")
	}
	if !sort.StringsAreSorted(diff1.RemovedMethods) {
		t.Error("RemovedMethods should be sorted")
	}
	if !sort.StringsAreSorted(diff1.AddedFields) {
		t.Error("AddedFields should be sorted")
	}
}

func TestDiffScans_Symmetry(t *testing.T) {
	result := loadRealAPK(t)
	dexFiles := loadTestDex(t)
	mutated := mutateResult(result)

	// old→new and new→old should be symmetric (added ↔ removed)
	forward := DiffScans(result, mutated, dexFiles, dexFiles, "", ScopeAll)
	reverse := DiffScans(mutated, result, dexFiles, dexFiles, "", ScopeAll)

	if len(forward.AddedMethods) != len(reverse.RemovedMethods) {
		t.Errorf("forward added=%d should equal reverse removed=%d",
			len(forward.AddedMethods), len(reverse.RemovedMethods))
	}
	if len(forward.RemovedMethods) != len(reverse.AddedMethods) {
		t.Errorf("forward removed=%d should equal reverse added=%d",
			len(forward.RemovedMethods), len(reverse.AddedMethods))
	}
	if len(forward.ChangedMethods) != len(reverse.ChangedMethods) {
		t.Errorf("forward changed=%d should equal reverse changed=%d",
			len(forward.ChangedMethods), len(reverse.ChangedMethods))
	}

	// Changed entries should have swapped old/new counts
	fMap := make(map[string]DiffEntry)
	for _, e := range forward.ChangedMethods {
		fMap[e.API] = e
	}
	for _, re := range reverse.ChangedMethods {
		fe, ok := fMap[re.API]
		if !ok {
			t.Errorf("reverse has changed %s not in forward", re.API)
			continue
		}
		if fe.OldCount != re.NewCount || fe.NewCount != re.OldCount {
			t.Errorf("symmetry broken for %s: forward(%d→%d) reverse(%d→%d)",
				re.API, fe.OldCount, fe.NewCount, re.OldCount, re.NewCount)
		}
	}
}

// --- Pure unit tests (no APK) ---

func TestDiffResult_Helpers(t *testing.T) {
	diff := &DiffResult{
		AddedMethods:   []string{"A", "B"},
		RemovedFields:  []string{"C"},
		ChangedMethods: []DiffEntry{{API: "D", OldCount: 1, NewCount: 2}},
	}

	if !diff.HasChanges() {
		t.Error("should have changes")
	}
	if diff.TotalAdded() != 2 {
		t.Errorf("TotalAdded = %d", diff.TotalAdded())
	}
	if diff.TotalRemoved() != 1 {
		t.Errorf("TotalRemoved = %d", diff.TotalRemoved())
	}
	if diff.TotalChanged() != 1 {
		t.Errorf("TotalChanged = %d", diff.TotalChanged())
	}
}

func TestDiffResult_Empty(t *testing.T) {
	diff := &DiffResult{}
	if diff.HasChanges() {
		t.Error("empty diff should not have changes")
	}
}
