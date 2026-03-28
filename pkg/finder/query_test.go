package finder

import (
	"testing"
)

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

	// Should match class descriptor
	if !m.matches("Landroid/location/LocationManager;->requestLocationUpdates()V") {
		t.Error("should match class methods")
	}
	if !m.matches("Landroid/location/LocationManager;->getLastKnownLocation()V") {
		t.Error("should match all methods of class")
	}
	if m.matches("Landroid/net/wifi/WifiManager;->getConnectionInfo()V") {
		t.Error("should not match other class")
	}
}

func TestQueryMatcherJavaMethodWithParams(t *testing.T) {
	m := newQueryMatcher("android.location.LocationManager#requestLocationUpdates(java.lang.String, long, float, android.location.LocationListener)")

	// Exact match should work
	target := "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
	if !m.matches(target) {
		t.Error("should match exact signature")
	}

	// Method name substring also in patterns
	if !m.matches("Landroid/location/LocationManager;->requestLocationUpdates(I)V") {
		t.Error("should match via class->method substring pattern")
	}

	// Different class should not match (no broad class pattern when params given)
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

	// Different overload should not match (exact DEX pattern)
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
		{"(java.lang.String, long, float, android.location.LocationListener)",
			"(Ljava/lang/String;JFLandroid/location/LocationListener;)V"},
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
	// Empty filter matches everything
	f := NewClassFilter(nil)
	if !f.Matches("Lcom/anything;") {
		t.Error("empty filter should match everything")
	}

	// With prefixes
	f2 := NewClassFilter([]string{"Lcom/myapp/", "Lcom/lib/"})
	if !f2.Matches("Lcom/myapp/Foo;") {
		t.Error("should match myapp prefix")
	}
	if !f2.Matches("Lcom/lib/Bar;") {
		t.Error("should match lib prefix")
	}
	if f2.Matches("Landroid/app/Activity;") {
		t.Error("should not match android prefix")
	}
}
