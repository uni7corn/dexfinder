package report

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"dex_method_finder/pkg/apk"
	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/mapping"
)

// --- Unit tests ---

func TestDexToJavaStacktrace(t *testing.T) {
	tests := []struct{ input, want string }{
		{"Lcom/example/Foo;->bar(Ljava/lang/String;)V", "com.example.Foo.bar(Foo.java)"},
		{"Lcom/example/Foo$Inner;->run()V", "com.example.Foo$Inner.run(Foo.java)"},
	}
	for _, tt := range tests {
		if got := dexToJavaStacktrace(tt.input); got != tt.want {
			t.Errorf("dexToJavaStacktrace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDexToJavaReadable(t *testing.T) {
	tests := []struct{ input, want string }{
		{"Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V",
			"android.location.LocationManager.requestLocationUpdates(String, long, float, LocationListener)"},
		{"Lcom/foo/Bar;->method()V", "com.foo.Bar.method()"},
		{"Lcom/foo/Bar;->method(I[Ljava/lang/String;)Z", "com.foo.Bar.method(int, String[])"},
	}
	for _, tt := range tests {
		if got := dexToJavaReadable(tt.input); got != tt.want {
			t.Errorf("dexToJavaReadable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDexParamsToJavaReadable(t *testing.T) {
	tests := []struct{ input, want string }{
		{"", ""},
		{"I", "int"},
		{"Ljava/lang/String;JFLandroid/location/LocationListener;", "String, long, float, LocationListener"},
		{"[I", "int[]"},
		{"[[Ljava/lang/String;", "String[][]"},
		{"CSFD", "char, short, float, double"},
	}
	for _, tt := range tests {
		if got := dexParamsToJavaReadable(tt.input); got != tt.want {
			t.Errorf("dexParamsToJavaReadable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShortName(t *testing.T) {
	if got := shortName("Lcom/example/Foo;->bar(I)V"); got != "Foo.bar(I)V" {
		t.Errorf("shortName = %q", got)
	}
	if got := shortName("plain"); got != "plain" {
		t.Errorf("shortName = %q", got)
	}
}

func TestDisplayConfigNilSafe(t *testing.T) {
	var dc *DisplayConfig
	if got := dc.FormatAPI("Lfoo;->bar()V"); got != "Lfoo;->bar()V" {
		t.Errorf("nil FormatAPI = %q", got)
	}
	if got := dc.FormatShort("Lfoo;->bar()V"); got != "foo.bar()V" {
		t.Errorf("nil FormatShort = %q", got)
	}
	if got := dc.FormatNode("Lfoo;->bar()V"); got != "foo.bar()V" {
		t.Errorf("nil FormatNode = %q", got)
	}
}

// --- Integration fixtures ---

func loadTestFixtures(t *testing.T) ([]*dex.DexFile, *finder.ScanResult, *mapping.ProguardMapping) {
	t.Helper()
	apkPath := "../../testdata/test.apk"
	mapPath := "../../testdata/test_mapping.txt"
	if _, err := os.Stat(apkPath); err != nil {
		t.Skip("test.apk not found")
	}
	dexFiles, err := apk.LoadDexFiles(apkPath)
	if err != nil {
		t.Fatalf("load APK: %v", err)
	}
	f := finder.NewDirectFinder(dexFiles, finder.NewClassFilter(nil), nil)
	result := f.Scan()
	var pm *mapping.ProguardMapping
	if _, err := os.Stat(mapPath); err == nil {
		pm, _ = mapping.LoadProguardMapping(mapPath)
	}
	return dexFiles, result, pm
}

// --- Text scan output assertions ---

func TestOutput_TextScan_ContainsTags(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	DumpScan(&buf, result, dexFiles, "requestLocationUpdates", finder.ScopeAll, &DisplayConfig{})
	out := buf.String()

	// Must contain [METHOD] tags
	methodCount := strings.Count(out, "[METHOD]")
	if methodCount != 3 {
		t.Errorf("text scan should contain 3 [METHOD] tags, got %d", methodCount)
	}

	// Each [METHOD] line should have the API signature
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "[METHOD]") {
			if !strings.Contains(line, "requestLocationUpdates") {
				t.Errorf("METHOD line should contain query: %s", line)
			}
			if !strings.Contains(line, "ref)") {
				t.Errorf("METHOD line should contain ref count: %s", line)
			}
		}
	}

	// Must contain indented caller lines
	if !strings.Contains(out, "       L") {
		t.Error("text scan should contain indented caller lines")
	}
}

func TestOutput_TextScan_MappingQuery(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	DumpScan(&buf, result, dexFiles, "KotlinCases", finder.ScopeAll, &DisplayConfig{Mapping: pm})
	if buf.Len() == 0 {
		t.Error("original name + mapping should produce output")
	}
	// Should contain at least some [METHOD] tags
	if !strings.Contains(buf.String(), "[METHOD]") {
		t.Error("output should contain [METHOD] tags")
	}
}

// --- Text trace: tree layout assertions ---

func TestOutput_TraceTree_Format(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()

	// Tree must contain tree connectors
	if !strings.Contains(out, "├──") && !strings.Contains(out, "└──") {
		t.Error("tree layout missing tree connectors (├── or └──)")
	}
	// Tree must NOT contain flat chain headers
	if strings.Contains(out, "Call chain #") {
		t.Error("tree layout should not contain 'Call chain #'")
	}
	// Java style: should contain .java) suffix
	if !strings.Contains(out, ".java)") {
		t.Error("java style should contain .java) in method names")
	}
}

func TestOutput_TraceTree_DexStyle(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleDex}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()

	// DEX style should contain type descriptors like )V or )I
	if !strings.Contains(out, ")V") && !strings.Contains(out, ")Z") && !strings.Contains(out, ")Ljava") {
		t.Error("dex style should contain DEX return type descriptors")
	}
	// Should still have tree connectors
	if !strings.Contains(out, "├──") && !strings.Contains(out, "└──") {
		t.Error("tree layout should have connectors even in dex style")
	}
}

// --- Text trace: list layout assertions ---

func TestOutput_TraceList_Format(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Layout: LayoutList, Style: StyleJava}
	DumpTrace(&buf, result, dexFiles, "requestLocationUpdates", 3, dc)
	out := buf.String()

	// List must contain chain headers
	chainCount := strings.Count(out, "--- Call chain #")
	if chainCount != 4 {
		t.Errorf("list should have 4 chain headers, got %d", chainCount)
	}
	// Must contain "at" lines
	if !strings.Contains(out, "\tat ") {
		t.Error("list layout should contain stacktrace 'at' lines")
	}
	// Must NOT contain tree connectors
	if strings.Contains(out, "├──") || strings.Contains(out, "└──") {
		t.Error("list layout should not contain tree connectors")
	}
}

func TestOutput_TraceList_ChainOrder(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Layout: LayoutList, Style: StyleJava}
	DumpTrace(&buf, result, dexFiles, "requestLocationUpdates", 3, dc)

	// Each chain should end with the target API (bottom of stack)
	chains := strings.Split(buf.String(), "--- Call chain #")
	for i, chain := range chains {
		if i == 0 {
			continue // skip header
		}
		lines := strings.Split(strings.TrimSpace(chain), "\n")
		lastAt := ""
		for _, line := range lines {
			if strings.HasPrefix(line, "\tat ") {
				lastAt = line
			}
		}
		if lastAt != "" && !strings.Contains(lastAt, "requestLocationUpdates") {
			t.Errorf("chain %d last 'at' line should be target API: %s", i, lastAt)
		}
	}
}

// --- show-obf assertions ---

func TestOutput_ShowObf_ContainsAnnotation(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava, ShowObf: true}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 1, dc)
	out := buf.String()

	obfCount := strings.Count(out, "[obf:")
	if obfCount == 0 {
		t.Error("show-obf should include [obf: ...] annotations")
	}
}

func TestOutput_NoShowObf_NoAnnotation(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava, ShowObf: false}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 1, dc)
	if strings.Contains(buf.String(), "[obf:") {
		t.Error("without show-obf, output should NOT contain [obf:] annotations")
	}
}

// --- JSON trace assertions ---

func TestOutput_JSONTree_Structure(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Layout: LayoutTree, Style: StyleJava}
	DumpTraceJSON(&buf, result, dexFiles, "requestLocationUpdates", 2, dc)

	var data struct {
		Targets []struct {
			API    string          `json:"api"`
			Tree   json.RawMessage `json:"tree"`
			Chains json.RawMessage `json:"chains"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(data.Targets) != 3 {
		t.Errorf("JSON tree should have 3 targets (3 overloads), got %d", len(data.Targets))
	}
	for i, tgt := range data.Targets {
		if tgt.Tree == nil {
			t.Errorf("target %d should have 'tree' field", i)
		}
		if tgt.Chains != nil {
			t.Errorf("target %d should NOT have 'chains' field in tree layout", i)
		}
		if tgt.API == "" {
			t.Errorf("target %d has empty API", i)
		}
	}
}

func TestOutput_JSONList_Structure(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Layout: LayoutList, Style: StyleJava}
	DumpTraceJSON(&buf, result, dexFiles, "requestLocationUpdates", 2, dc)

	var data struct {
		Targets []struct {
			API    string     `json:"api"`
			Tree   *struct{}  `json:"tree"`
			Chains [][]string `json:"chains"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	totalChains := 0
	for i, tgt := range data.Targets {
		if tgt.Tree != nil {
			t.Errorf("target %d should NOT have 'tree' in list layout", i)
		}
		totalChains += len(tgt.Chains)
		// Each chain should have ≥2 entries
		for j, chain := range tgt.Chains {
			if len(chain) < 2 {
				t.Errorf("target %d chain %d too short: %v", i, j, chain)
			}
		}
	}
	if totalChains != 4 {
		t.Errorf("JSON list total chains = %d, want 4", totalChains)
	}
}

func TestOutput_JSONShowObf(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("no mapping")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutList, Style: StyleJava, ShowObf: true}
	DumpTraceJSON(&buf, result, dexFiles, "KotlinCases", 2, dc)
	if !strings.Contains(buf.String(), "[obf:") {
		t.Error("JSON with show-obf should contain [obf:] in method names")
	}
}

// --- Edge cases ---

func TestOutput_NoResults(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Layout: LayoutTree, Style: StyleJava}
	DumpTrace(&buf, result, dexFiles, "NonExistentMethodXYZ12345", 2, dc)
	if !strings.Contains(buf.String(), "No matching APIs found") {
		t.Error("should say 'No matching APIs found'")
	}
}

func TestOutput_EmptyQuery(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpTrace(&buf, result, dexFiles, "", 2, dc)
	if !strings.Contains(buf.String(), "--query is required") {
		t.Error("should say '--query is required'")
	}
}
