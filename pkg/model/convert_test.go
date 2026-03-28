package model

import (
	"testing"
)

func TestParseDexParamTypes(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"I", []string{"I"}},
		{"IJ", []string{"I", "J"}},
		{"Ljava/lang/String;", []string{"Ljava/lang/String;"}},
		{"Ljava/lang/String;JFLandroid/location/LocationListener;",
			[]string{"Ljava/lang/String;", "J", "F", "Landroid/location/LocationListener;"}},
		{"[I", []string{"[I"}},
		{"[[Ljava/lang/String;", []string{"[[Ljava/lang/String;"}},
		{"ZBCSIJFD", []string{"Z", "B", "C", "S", "I", "J", "F", "D"}},
		{"[Ljava/lang/String;I", []string{"[Ljava/lang/String;", "I"}},
	}

	for _, tt := range tests {
		got := parseDexParamTypes(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseDexParamTypes(%q) = %v (len %d), want %v (len %d)",
				tt.input, got, len(got), tt.want, len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseDexParamTypes(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestDexToJavaReadable(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Lcom/foo/Bar;->method(I)V", "com.foo.Bar.method(...)"},
		{"Lcom/foo/Bar;", "com.foo.Bar"},
		{"Lcom/foo/Bar;->field:I", "com.foo.Bar.field:I"},
	}
	for _, tt := range tests {
		got := dexToJavaReadable(tt.input)
		if got != tt.want {
			t.Errorf("dexToJavaReadable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDexClassToJava(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Lcom/foo/Bar;", "com.foo.Bar"},
		{"Ljava/lang/String;", "java.lang.String"},
		{"LFoo;", "Foo"},
	}
	for _, tt := range tests {
		got := dexClassToJava(tt.input)
		if got != tt.want {
			t.Errorf("dexClassToJava(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
