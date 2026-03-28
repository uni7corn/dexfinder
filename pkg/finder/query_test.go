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
	if !m.matches("Landroid/location/LocationManager;->requestLocationUpdates(I)V") {
		t.Error("should match via class->method substring pattern")
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
		t.Error("should not match different overload for exact DEX sig")
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
	tests := []struct {
		input, want string
	}{
		{"(java.lang.String, long, float, android.location.LocationListener)", "(Ljava/lang/String;JFLandroid/location/LocationListener;)V"},
		{"(int)", "(I)V"},
		{"()", "()V"},
		{"(int[], java.lang.String[][])", "([I[[Ljava/lang/String;)V"},
		{"(boolean, byte, char, short, double)", "(ZBCSD)V"},
	}
	for _, tt := range tests {
		got := javaParamsToDex(tt.input)
		if got != tt.want {
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

// --- Integration tests: query × mapping × format × trace combinations ---

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

// --- Input × Mapping combinations ---

func TestCombo_ObfName_NoMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "Lbb;", ScopeAll)
	if len(qr.MatchedMethods) == 0 && len(qr.MatchedFields) == 0 {
		t.Error("obfuscated name without mapping should find results")
	}
}

func TestCombo_ObfName_WithMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "Lbb;", ScopeAll, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 && len(qr.MatchedFields) == 0 {
		t.Error("obfuscated name with mapping should find results")
	}
}

func TestCombo_OrigSimpleName_NoMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeAll)
	total := len(qr.MatchedMethods) + len(qr.MatchedFields) + len(qr.MatchedStrings)
	if total != 0 {
		t.Error("original name without mapping should not find obfuscated classes")
	}
}

func TestCombo_OrigSimpleName_WithMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeAll, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 {
		t.Error("original simple name with mapping should find results via obfuscation lookup")
	}
}

func TestCombo_OrigFullName_NoMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "com.test.dexfinder.kotlin.KotlinCases", ScopeAll)
	total := len(qr.MatchedMethods) + len(qr.MatchedFields) + len(qr.MatchedStrings)
	if total != 0 {
		t.Error("original full name without mapping should not find obfuscated classes")
	}
}

func TestCombo_OrigFullName_WithMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "com.test.dexfinder.kotlin.KotlinCases", ScopeAll, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 {
		t.Error("original full name with mapping should find results via obfuscation lookup")
	}
}

// --- Verify mapping doesn't break non-mapping queries ---

func TestCombo_FrameworkAPI_WithMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "requestLocationUpdates", ScopeAll, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 {
		t.Error("framework API query should still work with mapping loaded")
	}
}

// --- Scope combinations ---

func TestCombo_OrigName_ScopeCallee(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeCallee, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 {
		t.Error("callee scope with mapping should find results")
	}
	if len(qr.MatchedCallers) != 0 {
		t.Error("callee scope should not return caller matches")
	}
}

func TestCombo_OrigName_ScopeCaller(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "com.test.dexfinder.kotlin.KotlinCases", ScopeCaller, QueryOption{Mapping: pm})
	if len(qr.MatchedCallers) == 0 {
		t.Error("caller scope with mapping should find what KotlinCases calls internally")
	}
	if len(qr.MatchedMethods) != 0 {
		t.Error("caller scope should not return callee matches")
	}
}

func TestCombo_StringScope(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "android.app.ActivityThread", ScopeString)
	if len(qr.MatchedStrings) == 0 {
		t.Error("string scope should find ActivityThread string constant")
	}
}

func TestCombo_StringTableScope(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	qr := Query(result, dexFiles, "LocationListener", ScopeStringTable)
	if len(qr.MatchedStringTable) == 0 {
		t.Error("string-table scope should find LocationListener in DEX string table")
	}
}

// --- Trace combinations (call graph) ---

func TestCombo_Trace_ObfName_WithMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	// Trace uses ScopeCallee internally
	qr := Query(result, dexFiles, "LJ7;", ScopeCallee, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 {
		t.Error("trace with obf name + mapping should find results")
	}
	cg := BuildCallGraph(result, dexFiles)
	for api := range qr.MatchedMethods {
		tree := cg.TraceCallers(api, 3)
		if tree == nil {
			t.Errorf("TraceCallers returned nil for %s", api)
		}
		break
	}
}

func TestCombo_Trace_OrigName_WithMapping(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)
	qr := Query(result, dexFiles, "KotlinCases", ScopeCallee, QueryOption{Mapping: pm})
	if len(qr.MatchedMethods) == 0 {
		t.Error("trace with original name + mapping should find results")
	}
	cg := BuildCallGraph(result, dexFiles)
	chainCount := 0
	for api := range qr.MatchedMethods {
		tree := cg.TraceCallers(api, 3)
		chains := FlatCallerChains(tree, 50)
		chainCount += len(chains)
	}
	if chainCount == 0 {
		t.Error("expected at least one call chain")
	}
}

// --- Result consistency: same results regardless of query format ---

func TestCombo_ConsistentResults_ObfVsOrig(t *testing.T) {
	dexFiles, result := scanTestAPK(t)
	pm := loadMappingForTest(t)

	// Query by obfuscated name
	qr1 := Query(result, dexFiles, "LJ7;", ScopeCallee, QueryOption{Mapping: pm})
	// Query by original name
	qr2 := Query(result, dexFiles, "KotlinCases", ScopeCallee, QueryOption{Mapping: pm})

	// Both should find the same set of methods (maybe qr2 finds more due to inner classes)
	// At minimum, every method in qr1 should also be in qr2
	for api := range qr1.MatchedMethods {
		if _, ok := qr2.MatchedMethods[api]; !ok {
			// qr2 might match more (inner classes), but should at least contain LJ7; matches
			// Check if this specific key is LJ7;-prefixed
			if len(api) > 4 && api[:4] == "LJ7;" {
				t.Errorf("method %s found by obf query but not by original name query", api)
			}
		}
	}
}
