package mapping

import (
	"os"
	"path/filepath"
	"testing"
)

const testMapping = `# comment line
com.example.app.MainActivity -> a.a:
    android.widget.TextView titleView -> a
    void onCreate(android.os.Bundle) -> b
    1:5:java.lang.String getName():10:14 -> c
    void <init>() -> <init>
com.example.app.utils.Helper -> a.b:
    int count -> a
    void doWork(int,java.lang.String) -> a
    int compute(int[],java.lang.String[][]) -> b
com.example.app.MainActivity$InnerClass -> a.a$a:
    void run() -> a
`

func loadTestMapping(t *testing.T) *ProguardMapping {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "mapping.txt")
	if err := os.WriteFile(path, []byte(testMapping), 0644); err != nil {
		t.Fatal(err)
	}
	pm, err := LoadProguardMapping(path)
	if err != nil {
		t.Fatalf("LoadProguardMapping: %v", err)
	}
	return pm
}

func TestClassMapping(t *testing.T) {
	pm := loadTestMapping(t)

	// Forward
	if got := pm.ObfuscateClass("com.example.app.MainActivity"); got != "a.a" {
		t.Errorf("ObfuscateClass = %q, want %q", got, "a.a")
	}
	if got := pm.ObfuscateClass("com.example.app.utils.Helper"); got != "a.b" {
		t.Errorf("ObfuscateClass = %q, want %q", got, "a.b")
	}

	// Reverse
	if got := pm.DeobfuscateClass("a.a"); got != "com.example.app.MainActivity" {
		t.Errorf("DeobfuscateClass = %q, want %q", got, "com.example.app.MainActivity")
	}
	if got := pm.DeobfuscateClass("a.b"); got != "com.example.app.utils.Helper" {
		t.Errorf("DeobfuscateClass = %q, want %q", got, "com.example.app.utils.Helper")
	}

	// Inner class
	if got := pm.ObfuscateClass("com.example.app.MainActivity$InnerClass"); got != "a.a$a" {
		t.Errorf("inner class obfuscate = %q, want %q", got, "a.a$a")
	}

	// Unknown class passthrough
	if got := pm.DeobfuscateClass("z.z"); got != "z.z" {
		t.Errorf("unknown class = %q, want %q", got, "z.z")
	}
}

func TestDeobfuscateDexSignature(t *testing.T) {
	pm := loadTestMapping(t)

	tests := []struct {
		obf  string
		want string
	}{
		// Method
		{"La/a;->b(Landroid/os/Bundle;)V", "Lcom/example/app/MainActivity;->onCreate(Landroid/os/Bundle;)V"},
		// Method with line number stripping
		{"La/a;->c()Ljava/lang/String;", "Lcom/example/app/MainActivity;->getName()Ljava/lang/String;"},
		// Field
		{"La/a;->a:Landroid/widget/TextView;", "Lcom/example/app/MainActivity;->titleView:Landroid/widget/TextView;"},
		// Class only
		{"La/b;", "Lcom/example/app/utils/Helper;"},
		// Unknown method (class deobf, method stays)
		{"La/a;->unknown()V", "Lcom/example/app/MainActivity;->unknown()V"},
		// Completely unknown
		{"Lz/z;->foo()V", "Lz/z;->foo()V"},
	}

	for _, tt := range tests {
		t.Run(tt.obf, func(t *testing.T) {
			got := pm.DeobfuscateDexSignature(tt.obf)
			if got != tt.want {
				t.Errorf("DeobfuscateDexSignature(%q) = %q, want %q", tt.obf, got, tt.want)
			}
		})
	}
}

func TestObfuscateDexSignature(t *testing.T) {
	pm := loadTestMapping(t)

	tests := []struct {
		orig string
		want string
	}{
		// Method
		{"Lcom/example/app/MainActivity;->onCreate(Landroid/os/Bundle;)V", "La/a;->b(Landroid/os/Bundle;)V"},
		// Field
		{"Lcom/example/app/MainActivity;->titleView:Landroid/widget/TextView;", "La/a;->a:Landroid/widget/TextView;"},
		// Class only
		{"Lcom/example/app/utils/Helper;", "La/b;"},
	}

	for _, tt := range tests {
		t.Run(tt.orig, func(t *testing.T) {
			got := pm.ObfuscateDexSignature(tt.orig)
			if got != tt.want {
				t.Errorf("ObfuscateDexSignature(%q) = %q, want %q", tt.orig, got, tt.want)
			}
		})
	}
}

func TestMappingSize(t *testing.T) {
	pm := loadTestMapping(t)
	if got := pm.Size(); got != 3 {
		t.Errorf("Size = %d, want 3", got)
	}
}

func TestStripLineNumbers(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"void method(int)", "void method(int)"},
		{"1:5:void method(int):10:14", "void method(int)"},
		{"10:20:int foo(java.lang.String):100:110", "int foo(java.lang.String)"},
		{"void nolines()", "void nolines()"},
	}
	for _, tt := range tests {
		got := stripLineNumbers(tt.input)
		if got != tt.want {
			t.Errorf("stripLineNumbers(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestJavaTypeToDexType(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"int", "I"},
		{"void", "V"},
		{"boolean", "Z"},
		{"long", "J"},
		{"float", "F"},
		{"double", "D"},
		{"java.lang.String", "Ljava/lang/String;"},
		{"int[]", "[I"},
		{"java.lang.String[][]", "[[Ljava/lang/String;"},
	}
	for _, tt := range tests {
		got := javaTypeToDexType(tt.input)
		if got != tt.want {
			t.Errorf("javaTypeToDexType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
