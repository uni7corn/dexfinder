package hiddenapi

import (
	"testing"
)

func TestApiListFromNames(t *testing.T) {
	tests := []struct {
		names []string
		want  ApiList
		ok    bool
	}{
		{[]string{"blocked"}, Blocked, true},
		{[]string{"blacklist"}, Blocked, true},
		{[]string{"sdk"}, Sdk, true},
		{[]string{"whitelist"}, Sdk, true},
		{[]string{"unsupported"}, Unsupported, true},
		{[]string{"greylist"}, Unsupported, true},
		{[]string{"max-target-o"}, MaxTargetO, true},
		{[]string{"greylist-max-p"}, MaxTargetP, true},
		{[]string{"max-target-q"}, MaxTargetQ, true},
		{[]string{"max-target-r"}, MaxTargetR, true},
		{[]string{"max-target-s"}, MaxTargetS, true},
		// Multiple flags: most restrictive wins
		{[]string{"sdk", "blocked"}, Blocked, true},
		{[]string{"unsupported", "max-target-o"}, MaxTargetO, true},
		// Informational flags
		{[]string{"sdk", "system-api"}, Sdk, true},
		{[]string{"sdk", "test-api"}, Sdk, true},
		// Unknown
		{[]string{"unknown-flag"}, Invalid, false},
		// Empty
		{[]string{""}, Invalid, false},
		// Whitespace
		{[]string{" blocked "}, Blocked, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, ok := ApiListFromNames(tt.names)
			if got != tt.want || ok != tt.ok {
				t.Errorf("ApiListFromNames(%v) = (%v, %v), want (%v, %v)",
					tt.names, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestApiListString(t *testing.T) {
	if Blocked.String() != "blocked" {
		t.Errorf("Blocked.String() = %q", Blocked.String())
	}
	if Sdk.String() != "sdk" {
		t.Errorf("Sdk.String() = %q", Sdk.String())
	}
	if Invalid.String() != "invalid" {
		t.Errorf("Invalid.String() = %q", Invalid.String())
	}
}

func TestApiListMaxSdkVersion(t *testing.T) {
	if Blocked.GetMaxAllowedSdkVersion() != 0 {
		t.Error("Blocked should have max sdk 0")
	}
	if MaxTargetO.GetMaxAllowedSdkVersion() != 26 {
		t.Error("MaxTargetO should have max sdk 26")
	}
	if MaxTargetP.GetMaxAllowedSdkVersion() != 28 {
		t.Error("MaxTargetP should have max sdk 28")
	}
	if Sdk.GetMaxAllowedSdkVersion() != 9999 {
		t.Error("Sdk should have max sdk 9999")
	}
}

func TestApiListFilter(t *testing.T) {
	// Default filter: exclude SDK
	f := NewApiListFilter(nil)
	if f.Matches(Sdk) {
		t.Error("default filter should not match Sdk")
	}
	if !f.Matches(Blocked) {
		t.Error("default filter should match Blocked")
	}
	if !f.Matches(Unsupported) {
		t.Error("default filter should match Unsupported")
	}
	if f.Matches(Invalid) {
		t.Error("filter should never match Invalid")
	}

	// Custom exclude
	f2 := NewApiListFilter([]string{"blocked", "unsupported"})
	if f2.Matches(Blocked) {
		t.Error("custom filter should not match Blocked")
	}
	if f2.Matches(Unsupported) {
		t.Error("custom filter should not match Unsupported")
	}
	if !f2.Matches(MaxTargetO) {
		t.Error("custom filter should match MaxTargetO")
	}
}
