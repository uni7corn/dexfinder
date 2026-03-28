package mapping

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIntegrationMappingFile(t *testing.T) {
	mappingPath := "../../testdata/integration_mapping.txt"
	if _, err := os.Stat(mappingPath); err != nil {
		t.Skip("integration_mapping.txt not found")
	}

	pm, err := LoadProguardMapping(mappingPath)
	if err != nil {
		t.Fatalf("LoadProguardMapping: %v", err)
	}

	if pm.Size() == 0 {
		t.Fatal("mapping is empty")
	}

	// Test class mappings
	tests := []struct {
		orig string
		obf  string
	}{
		{"com.example.app.MainActivity", "a.a"},
		{"com.example.app.network.ApiClient", "a.b"},
		{"com.example.app.utils.Logger", "a.c"},
		{"com.example.app.location.LocationTracker", "a.d"},
		{"com.example.app.service.BackgroundService", "a.e"},
		{"com.example.app.location.LocationTracker$1", "a.d$a"},
		{"com.example.app.service.BackgroundService$Worker", "a.e$a"},
	}

	for _, tt := range tests {
		if got := pm.ObfuscateClass(tt.orig); got != tt.obf {
			t.Errorf("ObfuscateClass(%q) = %q, want %q", tt.orig, got, tt.obf)
		}
		if got := pm.DeobfuscateClass(tt.obf); got != tt.orig {
			t.Errorf("DeobfuscateClass(%q) = %q, want %q", tt.obf, got, tt.orig)
		}
	}

	// Test method deobfuscation
	methodTests := []struct {
		obfDex  string
		wantDex string
	}{
		{
			"La/a;->a(Landroid/os/Bundle;)V",
			"Lcom/example/app/MainActivity;->onCreate(Landroid/os/Bundle;)V",
		},
		{
			"La/d;->c(Ljava/lang/String;JFLandroid/location/LocationListener;)V",
			"Lcom/example/app/location/LocationTracker;->requestLocation(Ljava/lang/String;JFLandroid/location/LocationListener;)V",
		},
		{
			"La/e;->b(Landroid/content/Intent;II)V",
			"Lcom/example/app/service/BackgroundService;->onStartCommand(Landroid/content/Intent;II)V",
		},
	}

	for _, tt := range methodTests {
		got := pm.DeobfuscateDexSignature(tt.obfDex)
		if got != tt.wantDex {
			t.Errorf("DeobfuscateDexSignature(%q) = %q, want %q", tt.obfDex, got, tt.wantDex)
		}
	}

	// Test field deobfuscation
	fieldTests := []struct {
		obfDex  string
		wantDex string
	}{
		{
			"La/a;->a:Landroid/widget/TextView;",
			"Lcom/example/app/MainActivity;->mTitle:Landroid/widget/TextView;",
		},
		{
			"La/d;->a:Landroid/location/LocationManager;",
			"Lcom/example/app/location/LocationTracker;->locationManager:Landroid/location/LocationManager;",
		},
	}

	for _, tt := range fieldTests {
		got := pm.DeobfuscateDexSignature(tt.obfDex)
		if got != tt.wantDex {
			t.Errorf("DeobfuscateDexSignature(%q) = %q, want %q", tt.obfDex, got, tt.wantDex)
		}
	}
}

func TestIntegrationRoundTrip(t *testing.T) {
	mappingPath := "../../testdata/integration_mapping.txt"
	if _, err := os.Stat(mappingPath); err != nil {
		t.Skip("integration_mapping.txt not found")
	}

	pm, err := LoadProguardMapping(mappingPath)
	if err != nil {
		t.Fatal(err)
	}

	origSig := "Lcom/example/app/MainActivity;->onCreate(Landroid/os/Bundle;)V"
	obfSig := pm.ObfuscateDexSignature(origSig)
	restored := pm.DeobfuscateDexSignature(obfSig)

	if restored != origSig {
		t.Errorf("round-trip failed: %q → %q → %q", origSig, obfSig, restored)
	}
}

func TestIntegrationLargeMapping(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large_mapping.txt")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		origClass := fmt.Sprintf("com.example.gen.Class%d", i)
		obfClass := fmt.Sprintf("a%d.a", i)
		fmt.Fprintf(f, "%s -> %s:\n", origClass, obfClass)
		for j := 0; j < 5; j++ {
			fmt.Fprintf(f, "    void method%d(int,java.lang.String) -> m%d\n", j, j)
		}
		fmt.Fprintf(f, "    int field%d -> f\n", i)
	}
	f.Close()

	pm, err := LoadProguardMapping(path)
	if err != nil {
		t.Fatalf("LoadProguardMapping: %v", err)
	}

	if pm.Size() != 1000 {
		t.Errorf("expected 1000 classes, got %d", pm.Size())
	}

	if got := pm.ObfuscateClass("com.example.gen.Class500"); got != "a500.a" {
		t.Errorf("class 500 = %q", got)
	}
}
