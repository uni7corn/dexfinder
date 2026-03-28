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
		{"Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V",
			"android.location.LocationManager.requestLocationUpdates(LocationManager.java)"},
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
	tests := []struct{ input, want string }{
		{"Lcom/example/Foo;->bar(I)V", "Foo.bar(I)V"},
		{"plain_string", "plain_string"},
	}
	for _, tt := range tests {
		if got := shortName(tt.input); got != tt.want {
			t.Errorf("shortName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDisplayConfigNilSafe(t *testing.T) {
	var dc *DisplayConfig
	if got := dc.FormatAPI("Lfoo;->bar()V"); got != "Lfoo;->bar()V" {
		t.Errorf("nil dc FormatAPI = %q", got)
	}
	if got := dc.FormatShort("Lfoo;->bar()V"); got != "foo.bar()V" {
		t.Errorf("nil dc FormatShort = %q", got)
	}
}

// --- Integration tests: output format × layout × style × mapping ---

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

func TestOutput_TextScan_NoMapping(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpScan(&buf, result, dexFiles, "requestLocationUpdates", finder.ScopeAll, dc)
	out := buf.String()
	if !strings.Contains(out, "[METHOD]") {
		t.Error("text scan should contain [METHOD] tag")
	}
	if !strings.Contains(out, "requestLocationUpdates") {
		t.Error("text scan should contain the queried method")
	}
}

func TestOutput_TextScan_WithMapping(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm}
	DumpScan(&buf, result, dexFiles, "KotlinCases", finder.ScopeAll, dc)
	if buf.Len() == 0 {
		t.Error("text scan with mapping + original name should produce output")
	}
}

func TestOutput_TextTrace_Tree_Java(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()
	if !strings.Contains(out, "├") && !strings.Contains(out, "└") {
		t.Error("tree layout should contain tree connectors")
	}
	if strings.Contains(out, "Call chain #") {
		t.Error("tree layout should NOT contain flat chain headers")
	}
}

func TestOutput_TextTrace_List_Java(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutList, Style: StyleJava}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()
	if !strings.Contains(out, "Call chain #") {
		t.Error("list layout should contain 'Call chain #' headers")
	}
	if !strings.Contains(out, "\tat ") {
		t.Error("list layout should contain stacktrace-style 'at' lines")
	}
}

func TestOutput_TextTrace_Tree_Dex(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleDex}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()
	if !strings.Contains(out, "├") && !strings.Contains(out, "└") {
		t.Error("tree layout should contain tree connectors")
	}
}

func TestOutput_TextTrace_ShowObf(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava, ShowObf: true}
	DumpTrace(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()
	if !strings.Contains(out, "[obf:") {
		t.Error("show-obf should include [obf: ...] annotations")
	}
}

func TestOutput_JSONTrace_Tree(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutTree, Style: StyleJava}
	DumpTraceJSON(&buf, result, dexFiles, "KotlinCases", 2, dc)

	var data map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	targets, ok := data["targets"].([]interface{})
	if !ok || len(targets) == 0 {
		t.Error("JSON tree should have targets array")
	}
	first := targets[0].(map[string]interface{})
	if _, ok := first["tree"]; !ok {
		t.Error("JSON tree layout should have 'tree' field")
	}
	if _, ok := first["chains"]; ok {
		t.Error("JSON tree layout should NOT have 'chains' field")
	}
}

func TestOutput_JSONTrace_List(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutList, Style: StyleJava}
	DumpTraceJSON(&buf, result, dexFiles, "KotlinCases", 2, dc)

	var data map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	targets := data["targets"].([]interface{})
	first := targets[0].(map[string]interface{})
	if _, ok := first["chains"]; !ok {
		t.Error("JSON list layout should have 'chains' field")
	}
	if _, ok := first["tree"]; ok {
		t.Error("JSON list layout should NOT have 'tree' field")
	}
}

func TestOutput_JSONTrace_ShowObf(t *testing.T) {
	dexFiles, result, pm := loadTestFixtures(t)
	if pm == nil {
		t.Skip("mapping not available")
	}
	var buf bytes.Buffer
	dc := &DisplayConfig{Mapping: pm, Layout: LayoutList, Style: StyleJava, ShowObf: true}
	DumpTraceJSON(&buf, result, dexFiles, "KotlinCases", 2, dc)
	out := buf.String()
	if !strings.Contains(out, "[obf:") {
		t.Error("JSON with show-obf should include [obf: ...] in method names")
	}
}

func TestOutput_NoResults(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{Layout: LayoutTree, Style: StyleJava}
	DumpTrace(&buf, result, dexFiles, "NonExistentMethodXYZ12345", 2, dc)
	out := buf.String()
	if !strings.Contains(out, "No matching APIs found") {
		t.Error("should print 'No matching APIs found' for non-existent query")
	}
}

func TestOutput_EmptyQuery(t *testing.T) {
	dexFiles, result, _ := loadTestFixtures(t)
	var buf bytes.Buffer
	dc := &DisplayConfig{}
	DumpTrace(&buf, result, dexFiles, "", 2, dc)
	out := buf.String()
	if !strings.Contains(out, "--query is required") {
		t.Error("should print error for empty query in trace mode")
	}
}
