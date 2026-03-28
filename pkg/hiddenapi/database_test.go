package hiddenapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseLoadAndQuery(t *testing.T) {
	// Create temp CSV
	csv := `Landroid/app/Activity;->mCalled:Z,unsupported
Landroid/os/ServiceManager;->getService(Ljava/lang/String;)Landroid/os/IBinder;,blocked
Landroid/app/ActivityThread;->currentActivityThread()Landroid/app/ActivityThread;,max-target-o
Landroid/view/View;->mContext:Landroid/content/Context;,sdk
`
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "flags.csv")
	if err := os.WriteFile(csvPath, []byte(csv), 0644); err != nil {
		t.Fatal(err)
	}

	filter := NewApiListFilter(nil) // exclude SDK
	db := NewDatabase(filter)
	if err := db.LoadFromFile(csvPath); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	// Full signature lookup
	if got := db.GetApiList("Landroid/os/ServiceManager;->getService(Ljava/lang/String;)Landroid/os/IBinder;"); got != Blocked {
		t.Errorf("getService = %v, want Blocked", got)
	}

	// Class-only lookup
	if got := db.GetApiList("Landroid/os/ServiceManager;"); got == Invalid {
		t.Error("class-only lookup should find entry")
	}

	// Method-without-params lookup
	if got := db.GetApiList("Landroid/os/ServiceManager;->getService"); got == Invalid {
		t.Error("method-without-params lookup should find entry")
	}

	// ShouldReport: blocked should be reported (not SDK)
	if !db.ShouldReport("Landroid/os/ServiceManager;->getService(Ljava/lang/String;)Landroid/os/IBinder;") {
		t.Error("blocked API should be reported")
	}

	// ShouldReport: SDK should not be reported
	if db.ShouldReport("Landroid/view/View;->mContext:Landroid/content/Context;") {
		t.Error("SDK API should not be reported")
	}

	// Unknown signature
	if got := db.GetApiList("Lcom/unknown/Class;->method()V"); got != Invalid {
		t.Errorf("unknown signature = %v, want Invalid", got)
	}
}

func TestDatabaseSignatureSource(t *testing.T) {
	filter := NewApiListFilter(nil)
	db := NewDatabase(filter)

	db.AddSignatureSource("Landroid/app/Activity;", SourceBoot)
	db.AddSignatureSource("Lcom/myapp/MyClass;", SourceApp)

	if !db.IsInBoot("Landroid/app/Activity;") {
		t.Error("Activity should be in boot")
	}
	if !db.IsInBoot("Landroid/app/Activity;->onCreate(Landroid/os/Bundle;)V") {
		t.Error("Activity method should resolve to boot via class")
	}
	if db.IsInBoot("Lcom/myapp/MyClass;") {
		t.Error("MyClass should not be in boot")
	}
	if db.GetSignatureSource("Lcom/unknown/X;") != SourceUnknown {
		t.Error("unknown class should be SourceUnknown")
	}

	// Boot takes precedence
	db.AddSignatureSource("Lcom/myapp/MyClass;", SourceBoot)
	if !db.IsInBoot("Lcom/myapp/MyClass;") {
		t.Error("boot should take precedence")
	}
}

func TestToInternalName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"com.example.Foo", "Lcom/example/Foo;"},
		{"java.lang.String", "Ljava/lang/String;"},
		{"Foo", "LFoo;"},
	}
	for _, tt := range tests {
		if got := ToInternalName(tt.input); got != tt.want {
			t.Errorf("ToInternalName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMoreRestrictiveWins(t *testing.T) {
	csv := `Landroid/test/A;->foo()V,unsupported
Landroid/test/A;->foo()V,blocked
`
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "flags.csv")
	os.WriteFile(csvPath, []byte(csv), 0644)

	filter := NewApiListFilter(nil)
	db := NewDatabase(filter)
	db.LoadFromFile(csvPath)

	// Blocked is more restrictive, should win
	if got := db.GetApiList("Landroid/test/A;->foo()V"); got != Blocked {
		t.Errorf("expected Blocked, got %v", got)
	}
}
