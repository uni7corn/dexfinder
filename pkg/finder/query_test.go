package finder

import (
	"os"
	"testing"

	"dex_method_finder/pkg/apk"
	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/mapping"
)

// --- Unit tests for query matcher ---

func TestQueryMatcherSimpleName(t *testing.T) {
	m := newQueryMatcher("requestLocationUpdates")
	if !m.matches("Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JF)V") {
		t.Error("should match method containing the name")
	}
	if m.matches("Landroid/location/LocationManager;->getLastKnownLocation()V") {
		t.Error("should not match unrelated method")
	}
}

func TestQueryMatcherJavaClass(t *testing.T) {
	m := newQueryMatcher("android.location.LocationManager")
	if !m.matches("Landroid/location/LocationManager;->requestLocationUpdates()V") {
		t.Error("should match class methods")
	}
	if m.matches("Landroid/net/wifi/WifiManager;->getConnectionInfo()V") {
		t.Error("should not match other class")
	}
}

func TestQueryMatcherJavaMethodWithParams(t *testing.T) {
	m := newQueryMatcher("android.location.LocationManager#requestLocationUpdates(java.lang.String, long, float, android.location.LocationListener)")
	target := "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
	if !m.matches(target) {
		t.Error("should match exact signature")
	}
	if m.matches("Landroid/location/LocationManager;->getLastKnownLocation()V") {
		t.Error("should not match different method when params are specified")
	}
}

func TestQueryMatcherDexSignature(t *testing.T) {
	m := newQueryMatcher("Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V")
	target := "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
	if !m.matches(target) {
		t.Error("should match exact DEX signature")
	}
	other := "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;Landroid/os/Looper;)V"
	if m.matches(other) {
		t.Error("should not match different overload")
	}
}

func TestQueryMatcherCaseInsensitive(t *testing.T) {
	m := newQueryMatcher("locationmanager")
	if !m.matches("Landroid/location/LocationManager;->foo()V") {
		t.Error("should match case-insensitively")
	}
}

func TestQueryMatcherPartialPath(t *testing.T) {
	m := newQueryMatcher("location/LocationManager")
	if !m.matches("Landroid/location/LocationManager;->foo()V") {
		t.Error("should match partial path")
	}
}

func TestJavaParamsToDex(t *testing.T) {
	tests := []struct{ input, want string }{
		{"(java.lang.String, long, float, android.location.LocationListener)", "(Ljava/lang/String;JFLandroid/location/LocationListener;)V"},
		{"(int)", "(I)V"},
		{"()", "()V"},
		{"(int[], java.lang.String[][])", "([I[[Ljava/lang/String;)V"},
		{"(boolean, byte, char, short, double)", "(ZBCSD)V"},
	}
	for _, tt := range tests {
		if got := javaParamsToDex(tt.input); got != tt.want {
			t.Errorf("javaParamsToDex(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClassFilter(t *testing.T) {
	f := NewClassFilter(nil)
	if !f.Matches("Lcom/anything;") {
		t.Error("empty filter should match everything")
	}
	f2 := NewClassFilter([]string{"Lcom/myapp/", "Lcom/lib/"})
	if !f2.Matches("Lcom/myapp/Foo;") {
		t.Error("should match myapp prefix")
	}
	if f2.Matches("Landroid/app/Activity;") {
		t.Error("should not match android prefix")
	}
}

// --- Test fixtures ---

func loadMappingForTest(t *testing.T) *mapping.ProguardMapping {
	t.Helper()
	path := "../../testdata/test_mapping.txt"
	if _, err := os.Stat(path); err != nil {
		t.Skip("test_mapping.txt not found")
	}
	pm, err := mapping.LoadProguardMapping(path)
	if err != nil {
		t.Fatalf("load mapping: %v", err)
	}
	return pm
}

func scanTestAPK(t *testing.T) ([]*dex.DexFile, *ScanResult) {
	t.Helper()
	apkPath := "../../testdata/test.apk"
	if _, err := os.Stat(apkPath); err != nil {
		t.Skip("test.apk not found")
	}
	dexFiles, err := apk.LoadDexFiles(apkPath)
	if err != nil {
		t.Fatalf("load APK: %v", err)
	}
	f := NewDirectFinder(dexFiles, NewClassFilter(nil), nil)
	result := f.Scan()
	return dexFiles, result
}

// --- Input × Mapping: exact count assertions ---

func TestCombo_ObfName_NoMapping_Count(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "LJ7;", ScopeCallee)
	// LJ7; (KotlinCases) as callee should have exactly 2 method references
	if got := len(qr.MatchedMethods); got != 2 {
		t.Errorf("obf name LJ7 callee methods = %d, want 2", got)
	}
}

func TestCombo_OrigSimpleName_NoMapping_Empty(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeAll)
	total := len(qr.MatchedMethods) + len(qr.MatchedFields) + len(qr.MatchedStrings)
	if total != 0 {
		t.Errorf("original name without mapping should find 0 results, got %d", total)
	}
}

func TestCombo_OrigSimpleName_WithMapping_FindsAll(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeCallee, QueryOption{Mapping: pm})
	// Simple name matches KotlinCases + all inner classes ($$ExternalSyntheticLambda*, $LocationData, etc.)
	if got := len(qr.MatchedMethods); got < 10 {
		t.Errorf("original simple name + mapping should find >=10 methods (inner classes), got %d", got)
	}
}

func TestCombo_OrigFullName_WithMapping_Count(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "com.test.dexfinder.kotlin.KotlinCases", ScopeCallee, QueryOption{Mapping: pm})
	// Full name also matches inner classes via J7 prefix
	if got := len(qr.MatchedMethods); got < 2 {
		t.Errorf("original full name + mapping callee methods = %d, want >= 2", got)
	}
}

func TestCombo_OrigFullName_NoMapping_Empty(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "com.test.dexfinder.kotlin.KotlinCases", ScopeAll)
	total := len(qr.MatchedMethods) + len(qr.MatchedFields) + len(qr.MatchedStrings)
	if total != 0 {
		t.Errorf("original full name without mapping should find 0, got %d", total)
	}
}

// --- Framework API not broken by mapping ---

func TestCombo_FrameworkAPI_WithMapping_Count(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "requestLocationUpdates", ScopeCallee, QueryOption{Mapping: pm})
	// Should find exactly 3 requestLocationUpdates overloads
	if got := len(qr.MatchedMethods); got != 3 {
		t.Errorf("requestLocationUpdates callee methods = %d, want 3", got)
	}
}

func TestCombo_FrameworkAPI_NoMapping_SameCount(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr1 := Query(result, dexFiles, "requestLocationUpdates", ScopeCallee)
	pm := loadMappingForTest(t)
	qr2 := Query(result, dexFiles, "requestLocationUpdates", ScopeCallee, QueryOption{Mapping: pm})
	if len(qr1.MatchedMethods) != len(qr2.MatchedMethods) {
		t.Errorf("mapping should not change framework API count: %d vs %d",
			len(qr1.MatchedMethods), len(qr2.MatchedMethods))
	}
}

// --- Scope assertions ---

func TestCombo_ScopeCallee_NoCallerResults(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeCallee, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 {
		t.Error("callee scope should find methods")
	}
	if len(qr.MatchedCallers) != 0 {
		t.Errorf("callee scope should have 0 caller results, got %d", len(qr.MatchedCallers))
	}
}

func TestCombo_ScopeCaller_NoCalleeResults(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "com.test.dexfinder.kotlin.KotlinCases", ScopeCaller, QueryOption{Mapping: pm})
	if len(qr.MatchedCallers) == 0 {
		t.Error("caller scope should find what KotlinCases calls")
	}
	if len(qr.MatchedMethods) != 0 {
		t.Errorf("caller scope should have 0 callee results, got %d", len(qr.MatchedMethods))
	}
}

func TestCombo_ScopeString_Content(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "android.app.ActivityThread", ScopeString)
	if got := len(qr.MatchedStrings); got != 1 {
		t.Errorf("string scope for ActivityThread should find 1 string, got %d", got)
	}
	// Verify the string value
	for str := range qr.MatchedStrings {
		if str != "android.app.ActivityThread" {
			t.Errorf("matched string = %q, want 'android.app.ActivityThread'", str)
		}
	}
	// Should not have method or field results
	if len(qr.MatchedMethods) != 0 {
		t.Errorf("string scope should have 0 methods, got %d", len(qr.MatchedMethods))
	}
}

func TestCombo_ScopeStringTable_FindsMoreThanCode(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr1 := Query(result, dexFiles, "LocationListener", ScopeString)
	qr2 := Query(result, dexFiles, "LocationListener", ScopeString|ScopeStringTable)
	// String table should find entries not in code (class descriptors, annotations)
	if len(qr2.MatchedStringTable) == 0 {
		t.Error("string-table scope should find additional entries beyond code strings")
	}
	_ = qr1 // qr1 may or may not have code strings
}

func TestCombo_ScopeAll_NoCallerByDefault(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeAll, QueryOption{Mapping: pm})
	if len(qr.MatchedCallers) != 0 {
		t.Errorf("scope=all should NOT include caller results, got %d", len(qr.MatchedCallers))
	}
}

// --- Trace: chain count assertions ---

func TestCombo_TraceChainCount(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "requestLocationUpdates", ScopeCallee)
	cg := BuildCallGraph(result, dexFiles)
	totalChains := 0
	for api := range qr.MatchedMethods {
		tree := cg.TraceCallers(api, 3)
		chains := FlatCallerChains(tree, 100)
		totalChains += len(chains)
	}
	// 3 overloads of requestLocationUpdates, should produce exactly 4 chains at depth 3
	if totalChains != 4 {
		t.Errorf("trace chain count = %d, want 4", totalChains)
	}
}

func TestCombo_TraceChainContent(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "requestLocationUpdates", ScopeCallee)
	cg := BuildCallGraph(result, dexFiles)

	// Every chain should start with the target API and end with a caller
	for api := range qr.MatchedMethods {
		tree := cg.TraceCallers(api, 5)
		chains := FlatCallerChains(tree, 100)
		for i, chain := range chains {
			if len(chain) == 0 {
				t.Errorf("chain %d is empty", i)
				continue
			}
			// chain[0] = target API
			if chain[0] != api {
				t.Errorf("chain %d should start with %s, got %s", i, api, chain[0])
			}
			// chain should have at least 2 entries (target + 1 caller)
			if len(chain) < 2 {
				t.Errorf("chain %d too short: %v", i, chain)
			}
		}
	}
}

func TestCombo_TraceDepthRespected(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "requestLocationUpdates", ScopeCallee)
	cg := BuildCallGraph(result, dexFiles)

	for api := range qr.MatchedMethods {
		tree := cg.TraceCallers(api, 2)
		chains := FlatCallerChains(tree, 100)
		for _, chain := range chains {
			// depth=2 means max 3 entries (target + 2 levels of callers)
			if len(chain) > 3 {
				t.Errorf("chain exceeds depth 2: %d entries: %v", len(chain), chain)
			}
		}
	}
}

// --- Consistency: obf vs original name produce same underlying results ---

func TestCombo_ConsistentResults_ObfVsOrigFull(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)

	// KotlinCases -> J7
	// LJ7; direct match
	qr1 := Query(result, dexFiles, "LJ7;", ScopeCallee, QueryOption{Mapping: pm})
	// Full original name
	qr2 := Query(result, dexFiles, "com.test.dexfinder.kotlin.KotlinCases", ScopeCallee, QueryOption{Mapping: pm})

	// LJ7; exact matches should be a subset of full name matches
	for api := range qr1.MatchedMethods {
		if _, ok := qr2.MatchedMethods[api]; !ok {
			t.Errorf("method %s found by LJ7; but not by full original name", api)
		}
	}
}
