package report

import (
	"testing"
)

func TestDexToJavaStacktrace(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{
			"Lcom/example/Foo;->bar(Ljava/lang/String;)V",
			"com.example.Foo.bar(Foo.java)",
		},
		{
			"Lcom/example/Foo$Inner;->run()V",
			"com.example.Foo$Inner.run(Foo.java)",
		},
		{
			"Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V",
			"android.location.LocationManager.requestLocationUpdates(LocationManager.java)",
		},
	}

	for _, tt := range tests {
		got := dexToJavaStacktrace(tt.input)
		if got != tt.want {
			t.Errorf("dexToJavaStacktrace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDexToJavaReadable(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{
			"Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V",
			"android.location.LocationManager.requestLocationUpdates(String, long, float, LocationListener)",
		},
		{
			"Lcom/foo/Bar;->method()V",
			"com.foo.Bar.method()",
		},
		{
			"Lcom/foo/Bar;->method(I[Ljava/lang/String;)Z",
			"com.foo.Bar.method(int, String[])",
		},
	}

	for _, tt := range tests {
		got := dexToJavaReadable(tt.input)
		if got != tt.want {
			t.Errorf("dexToJavaReadable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDexParamsToJavaReadable(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", ""},
		{"I", "int"},
		{"IJ", "int, long"},
		{"Ljava/lang/String;", "String"},
		{"Ljava/lang/String;JFLandroid/location/LocationListener;", "String, long, float, LocationListener"},
		{"[I", "int[]"},
		{"[[Ljava/lang/String;", "String[][]"},
		{"ZB", "boolean, byte"},
		{"CSFD", "char, short, float, double"},
	}

	for _, tt := range tests {
		got := dexParamsToJavaReadable(tt.input)
		if got != tt.want {
			t.Errorf("dexParamsToJavaReadable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShortName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Lcom/example/Foo;->bar(I)V", "Foo.bar(I)V"},
		{"Lcom/a/b/c/Baz;->qux()Ljava/lang/String;", "Baz.qux()Ljava/lang/String;"},
		{"plain_string", "plain_string"},
	}

	for _, tt := range tests {
		got := shortName(tt.input)
		if got != tt.want {
			t.Errorf("shortName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDisplayConfigNilSafe(t *testing.T) {
	var dc *DisplayConfig

	// Should not panic with nil config
	if got := dc.FormatAPI("Lfoo;->bar()V"); got != "Lfoo;->bar()V" {
		t.Errorf("nil dc FormatAPI = %q", got)
	}
	if got := dc.FormatShort("Lfoo;->bar()V"); got != "foo.bar()V" {
		t.Errorf("nil dc FormatShort = %q", got)
	}
}
