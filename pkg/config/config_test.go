package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".dexfinder.yaml")
	content := `# Test config
dex-file: app.apk
query: getDeviceId
format: json
layout: tree
style: java
mapping: mapping.txt
show-obf: true
api-flags: hiddenapi-flags.csv
class-filter: "Lcom/mycompany/"
trace: true
depth: 8
scope: callee
color: never
fail-on: blocked
output: result.json
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	tests := []struct{ field, got, want string }{
		{"DexFile", cfg.DexFile, "app.apk"},
		{"Query", cfg.Query, "getDeviceId"},
		{"Format", cfg.Format, "json"},
		{"Layout", cfg.Layout, "tree"},
		{"Style", cfg.Style, "java"},
		{"Mapping", cfg.Mapping, "mapping.txt"},
		{"ApiFlags", cfg.ApiFlags, "hiddenapi-flags.csv"},
		{"ClassFilter", cfg.ClassFilter, "Lcom/mycompany/"},
		{"Scope", cfg.Scope, "callee"},
		{"Color", cfg.Color, "never"},
		{"FailOn", cfg.FailOn, "blocked"},
		{"Output", cfg.Output, "result.json"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.field, tt.got, tt.want)
		}
	}

	if !cfg.ShowObf {
		t.Error("ShowObf should be true")
	}
	if !cfg.Trace {
		t.Error("Trace should be true")
	}
	if cfg.Depth != 8 {
		t.Errorf("Depth = %d, want 8", cfg.Depth)
	}
}

func TestParseConfigFile_Comments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `# comment line
dex-file: test.apk
# another comment
query: hello
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DexFile != "test.apk" {
		t.Errorf("DexFile = %q", cfg.DexFile)
	}
	if cfg.Query != "hello" {
		t.Errorf("Query = %q", cfg.Query)
	}
}

func TestParseConfigFile_QuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `class-filter: "Lcom/mycompany/,Lcom/mylib/"
query: 'getDeviceId'
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClassFilter != "Lcom/mycompany/,Lcom/mylib/" {
		t.Errorf("ClassFilter = %q", cfg.ClassFilter)
	}
	if cfg.Query != "getDeviceId" {
		t.Errorf("Query = %q", cfg.Query)
	}
}

func TestParseConfigFile_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	os.WriteFile(path, []byte(""), 0644)

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DexFile != "" {
		t.Error("empty config should have empty DexFile")
	}
}

func TestParseConfigFile_BoolValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	os.WriteFile(path, []byte("show-obf: yes\ntrace: 1\n"), 0644)

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ShowObf {
		t.Error("'yes' should parse as true")
	}
	if !cfg.Trace {
		t.Error("'1' should parse as true")
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// In a temp dir with no config file
	old, _ := os.Getwd()
	defer os.Chdir(old)
	dir := t.TempDir()
	os.Chdir(dir)

	cfg := Load()
	if cfg.DexFile != "" {
		t.Error("no config should return empty")
	}
}

func TestLoad_FindsConfigInParent(t *testing.T) {
	old, _ := os.Getwd()
	defer os.Chdir(old)

	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	os.Mkdir(child, 0755)
	os.WriteFile(filepath.Join(parent, ".dexfinder.yaml"), []byte("dex-file: found.apk\n"), 0644)
	os.Chdir(child)

	cfg := Load()
	if cfg.DexFile != "found.apk" {
		t.Errorf("should find config in parent, got DexFile=%q", cfg.DexFile)
	}
}

func TestApplyToFlags(t *testing.T) {
	cfg := &Config{
		DexFile: "app.apk",
		Query:   "hello",
		Depth:   10,
		Trace:   true,
	}

	dexFile := ""
	query := ""
	depth := 5
	trace := false

	flagSet := map[string]bool{"query": true} // query was explicitly set
	cfg.ApplyToFlags(flagSet,
		map[string]*string{"dex-file": &dexFile, "query": &query},
		map[string]*bool{"trace": &trace},
		map[string]*int{"depth": &depth},
	)

	if dexFile != "app.apk" {
		t.Errorf("dex-file should be set from config, got %q", dexFile)
	}
	if query != "" {
		t.Error("query was explicitly set, should not be overridden")
	}
	if depth != 10 {
		t.Errorf("depth should be 10, got %d", depth)
	}
	if !trace {
		t.Error("trace should be true")
	}
}

func TestParseBool(t *testing.T) {
	trueVals := []string{"true", "True", "TRUE", "yes", "Yes", "1"}
	for _, v := range trueVals {
		if !parseBool(v) {
			t.Errorf("parseBool(%q) should be true", v)
		}
	}
	falseVals := []string{"false", "no", "0", ""}
	for _, v := range falseVals {
		if parseBool(v) {
			t.Errorf("parseBool(%q) should be false", v)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct{ in string; want int }{
		{"0", 0},
		{"5", 5},
		{"123", 123},
		{"abc", 0},
		{"12abc", 12},
	}
	for _, tt := range tests {
		if got := parseInt(tt.in); got != tt.want {
			t.Errorf("parseInt(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
