package finder

import (
	"os"
	"strings"
	"testing"

	"dex_method_finder/pkg/apk"
	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/hiddenapi"
)

// findTestAPK looks for a real APK to test with.
func findTestAPK(t *testing.T) string {
	t.Helper()
	candidates := []string{
		os.Getenv("TEST_APK"),
		"../../testdata/test.apk",
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		candidates = append(candidates, home+"/Downloads/AppSearch.apk")
	}
	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("No test APK found. Set TEST_APK env var or place an APK in testdata/test.apk")
	return ""
}

func findTestMapping(t *testing.T) string {
	t.Helper()
	path := "../../testdata/test_mapping.txt"
	if _, err := os.Stat(path); err != nil {
		t.Skip("test_mapping.txt not found")
	}
	return path
}

func findHiddenAPICSV(t *testing.T) string {
	t.Helper()
	path := "../../testdata/hiddenapi-flags.csv"
	if _, err := os.Stat(path); err != nil {
		t.Skip("hiddenapi-flags.csv not found")
	}
	return path
}

func loadTestDex(t *testing.T) []*dex.DexFile {
	t.Helper()
	apkPath := findTestAPK(t)
	dexFiles, err := apk.LoadDexFiles(apkPath)
	if err != nil {
		t.Fatalf("LoadDexFiles: %v", err)
	}
	return dexFiles
}

func scanAll(t *testing.T, dexFiles []*dex.DexFile) *ScanResult {
	t.Helper()
	filter := NewClassFilter(nil)
	f := NewDirectFinder(dexFiles, filter, nil)
	return f.Scan()
}

// --- Basic Scan Tests ---

func TestIntegrationLoadAndScan(t *testing.T) {
	dexFiles := loadTestDex(t)
	if len(dexFiles) == 0 {
		t.Fatal("expected at least 1 DEX file")
	}
	for i, df := range dexFiles {
		if df.NumStringIDs() == 0 {
			t.Errorf("dex[%d]: no strings", i)
		}
		if df.NumTypeIDs() == 0 {
			t.Errorf("dex[%d]: no types", i)
		}
	}

	result := scanAll(t, dexFiles)
	if len(result.MethodRefs) == 0 {
		t.Error("expected method references")
	}
	if len(result.FieldRefs) == 0 {
		t.Error("expected field references")
	}
	if len(result.AllStrings) == 0 {
		t.Error("expected string table entries")
	}
}

// --- Obfuscation Tests ---

func TestIntegrationObfuscatedClassesExist(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	// R8 obfuscated APK should have short class names like "La;", "Lb;", etc.
	shortClassCount := 0
	for cls := range result.Classes {
		if len(cls) <= 5 && strings.HasPrefix(cls, "L") && strings.HasSuffix(cls, ";") {
			shortClassCount++
		}
	}
	if shortClassCount == 0 {
		t.Error("expected obfuscated short class names in R8 output")
	}
	t.Logf("Found %d short (obfuscated) class names", shortClassCount)
}

func TestIntegrationKeptClassesNotObfuscated(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	// ProGuard rules keep these classes
	keptClasses := []string{
		"Lcom/test/dexfinder/MainActivity;",
		"Lcom/test/dexfinder/TestEntry;",
		"Lcom/test/dexfinder/service/BackgroundLocationService;",
	}
	for _, cls := range keptClasses {
		if !result.Classes[cls] {
			t.Errorf("kept class %s not found in type_ids", cls)
		}
	}
}

func TestIntegrationR8InlinedMethods(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	// R8 with aggressive optimization inlines LocationCases/DeepCallChain into TestEntry
	// So requestLocationUpdates callers should include TestEntry directly
	refs := result.MethodRefs["Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"]
	if len(refs) == 0 {
		t.Fatal("requestLocationUpdates not found")
	}

	callers := make(map[string]bool)
	for _, ref := range refs {
		name := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
		callers[name] = true
	}

	// TestEntry should be a direct caller (R8 inlined LocationCases into it)
	if !callers["Lcom/test/dexfinder/TestEntry;->runAllTests(Landroid/content/Context;)V"] {
		t.Error("expected TestEntry.runAllTests as caller (R8 inline)")
	}
	// BackgroundLocationService should also be a direct caller
	found := false
	for name := range callers {
		if strings.Contains(name, "BackgroundLocationService") {
			found = true
		}
	}
	if !found {
		t.Error("expected BackgroundLocationService as caller")
	}
	t.Logf("requestLocationUpdates callers: %v", callers)
}

// --- Query Tests ---

func TestIntegrationQueryJavaStyle(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	qr := Query(result, dexFiles, "android.location.LocationManager", ScopeCallee)
	if len(qr.MatchedMethods) == 0 {
		t.Error("Java-style class query should match LocationManager methods")
	}
}

func TestIntegrationQueryDexStyle(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	qr := Query(result, dexFiles,
		"Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V",
		ScopeCallee)
	if len(qr.MatchedMethods) != 1 {
		t.Errorf("exact DEX query should match 1 method, got %d", len(qr.MatchedMethods))
	}
}

func TestIntegrationQueryReflection(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	// Class.forName should be found (our test app uses it in ReflectionCases)
	qr := Query(result, dexFiles, "forName", ScopeCallee)
	if len(qr.MatchedMethods) == 0 {
		t.Error("expected Class.forName references")
	}
}

func TestIntegrationQueryStrings(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	// Our test app reflects on "android.app.ActivityThread"
	qr := Query(result, dexFiles, "android.app.ActivityThread", ScopeString)
	if len(qr.MatchedStrings) == 0 {
		t.Error("expected string constant 'android.app.ActivityThread'")
	}
}

func TestIntegrationQueryStringTable(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	// String table should contain class descriptors even if not in const-string
	qr := Query(result, dexFiles, "LocationListener", ScopeStringTable)
	if len(qr.MatchedStringTable) == 0 {
		t.Error("expected 'LocationListener' in string table")
	}
}

// --- Call Graph Tests ---

func TestIntegrationCallGraph(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)

	cg := BuildCallGraph(result, dexFiles)

	// requestLocationUpdates should have callers
	api := "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
	callers := cg.GetDirectCallers(api)
	if len(callers) == 0 {
		t.Error("expected callers for requestLocationUpdates")
	}
}

func TestIntegrationTraceDepth(t *testing.T) {
	dexFiles := loadTestDex(t)
	result := scanAll(t, dexFiles)
	cg := BuildCallGraph(result, dexFiles)

	api := "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
	tree := cg.TraceCallers(api, 10)
	chains := FlatCallerChains(tree, 100)
	if len(chains) == 0 {
		t.Error("expected at least one call chain")
	}

	// Verify chain starts with the target API
	for _, chain := range chains {
		if chain[0] != api {
			t.Errorf("chain should start with target API, got %s", chain[0])
		}
	}
}

// --- Class Data & Instruction Decode Tests ---

func TestIntegrationClassData(t *testing.T) {
	dexFiles := loadTestDex(t)
	df := dexFiles[0]
	classCount := 0
	methodCount := 0

	for ci := range df.ClassDefs {
		cd := &df.ClassDefs[ci]
		classData := df.GetClassData(cd)
		if classData == nil {
			continue
		}
		classCount++
		for _, method := range classData.AllMethods() {
			methodCount++
			if method.CodeOff != 0 {
				codeItem := df.GetCodeItem(method.CodeOff)
				if codeItem == nil {
					t.Errorf("nil code item for method %d", method.MethodIdx)
					continue
				}
				// Decode all instructions without panic
				instructions := dex.DecodeAll(codeItem.Insns)
				if len(instructions) == 0 {
					t.Errorf("no instructions for method %d", method.MethodIdx)
				}
			}
		}
	}
	if classCount == 0 {
		t.Error("no classes with data")
	}
	t.Logf("Parsed %d classes, %d methods", classCount, methodCount)
}

// --- Hidden API Tests ---

func TestIntegrationHiddenAPI(t *testing.T) {
	csvPath := findHiddenAPICSV(t)
	dexFiles := loadTestDex(t)

	filter := hiddenapi.NewApiListFilter(nil)
	db := hiddenapi.NewDatabase(filter)
	if err := db.LoadFromFile(csvPath); err != nil {
		t.Fatalf("load CSV: %v", err)
	}

	f := NewDirectFinder(dexFiles, NewClassFilter(nil), db)
	result := f.Scan()
	filtered := result.FilterHiddenAPIs(db)

	// Our test app uses some hidden APIs via reflection
	reflections := result.FindPotentialReflection(db)
	t.Logf("Linking: %d methods, %d fields; Reflection: %d",
		len(filtered.MethodRefs), len(filtered.FieldRefs), len(reflections))

	// Should find at least some hidden API usage
	total := len(filtered.MethodRefs) + len(filtered.FieldRefs) + len(reflections)
	if total == 0 {
		t.Error("expected some hidden API findings in test APK")
	}
}

func TestIntegrationReflectionConsistency(t *testing.T) {
	csvPath := findHiddenAPICSV(t)
	dexFiles := loadTestDex(t)

	filter := hiddenapi.NewApiListFilter(nil)
	db := hiddenapi.NewDatabase(filter)
	db.LoadFromFile(csvPath)

	f := NewDirectFinder(dexFiles, NewClassFilter(nil), db)
	result := f.Scan()

	reflections := result.FindPotentialReflection(db)
	// Every reflection result should have a non-empty signature and string refs
	for _, ref := range reflections {
		if ref.Signature == "" {
			t.Error("empty signature in reflection result")
		}
		if ref.Class == "" {
			t.Error("empty class in reflection result")
		}
		if ref.Member == "" {
			t.Error("empty member in reflection result")
		}
		if !strings.Contains(ref.Signature, "->") {
			t.Errorf("signature should contain '->': %s", ref.Signature)
		}
	}
}
