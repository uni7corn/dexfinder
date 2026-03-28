package apk

import (
	"os"
	"testing"
)

func TestIsZip(t *testing.T) {
	if !isZip([]byte("PK\x03\x04rest")) {
		t.Error("should detect ZIP magic")
	}
	if isZip([]byte("dex\nrest")) {
		t.Error("DEX should not be ZIP")
	}
	if isZip([]byte{0x01}) {
		t.Error("short data should not be ZIP")
	}
}

func TestIsDex(t *testing.T) {
	if !isDex([]byte("dex\n035\x00rest")) {
		t.Error("should detect DEX magic")
	}
	if isDex([]byte("PK\x03\x04")) {
		t.Error("ZIP should not be DEX")
	}
}

func TestIsDexFileName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"classes.dex", true},
		{"classes2.dex", true},
		{"classes10.dex", true},
		{"other.dex", false},
		{"classes.txt", false},
	}

	for _, tt := range tests {
		if got := isDexFileName(tt.name); got != tt.want {
			t.Errorf("isDexFileName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := LoadDexFiles("/nonexistent/path.apk")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadBadFile(t *testing.T) {
	tmpFile := t.TempDir() + "/bad.apk"
	os.WriteFile(tmpFile, []byte("not a dex or zip"), 0644)
	_, err := LoadDexFiles(tmpFile)
	if err == nil {
		t.Error("expected error for bad file")
	}
}
