package dex

import (
	"testing"
)

func TestLEB128(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		want   uint32
		wantN  int
	}{
		{"zero", []byte{0x00}, 0, 1},
		{"one", []byte{0x01}, 1, 1},
		{"127", []byte{0x7F}, 127, 1},
		{"128", []byte{0x80, 0x01}, 128, 2},
		{"300", []byte{0xAC, 0x02}, 300, 2},
		{"16256", []byte{0x80, 0x7F}, 16256, 2},
		{"large", []byte{0xE5, 0x8E, 0x26}, 624485, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n := decodeULEB128(tt.data)
			if got != tt.want || n != tt.wantN {
				t.Errorf("decodeULEB128(%v) = (%d, %d), want (%d, %d)", tt.data, got, n, tt.want, tt.wantN)
			}
		})
	}
}

func TestSLEB128(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		want   int32
		wantN  int
	}{
		{"zero", []byte{0x00}, 0, 1},
		{"one", []byte{0x01}, 1, 1},
		{"minus1", []byte{0x7F}, -1, 1},
		{"minus128", []byte{0x80, 0x7F}, -128, 2},
		{"positive_two_byte", []byte{0x80, 0x01}, 128, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n := decodeSLEB128(tt.data)
			if got != tt.want || n != tt.wantN {
				t.Errorf("decodeSLEB128(%v) = (%d, %d), want (%d, %d)", tt.data, got, n, tt.want, tt.wantN)
			}
		})
	}
}

func TestLE16(t *testing.T) {
	data := []byte{0x34, 0x12, 0xCD, 0xAB}
	if got := le16(data, 0); got != 0x1234 {
		t.Errorf("le16(0) = 0x%04X, want 0x1234", got)
	}
	if got := le16(data, 2); got != 0xABCD {
		t.Errorf("le16(2) = 0x%04X, want 0xABCD", got)
	}
}

func TestLE32(t *testing.T) {
	data := []byte{0x78, 0x56, 0x34, 0x12}
	if got := le32(data, 0); got != 0x12345678 {
		t.Errorf("le32(0) = 0x%08X, want 0x12345678", got)
	}
}

// buildMinimalDex creates a minimal valid DEX file for testing.
func buildMinimalDex() []byte {
	// Minimal DEX with header only, 0 entries in all tables
	data := make([]byte, 112)
	copy(data[0:], "dex\n035\x00")     // magic
	// checksum at 8 (skip)
	// signature at 12 (skip)
	putLE32(data, 32, 112)              // file_size
	putLE32(data, 36, 0x70)             // header_size = 112
	putLE32(data, 40, 0x12345678)       // endian_tag
	// All sizes = 0, all offsets = 0
	return data
}

func putLE32(data []byte, off int, val uint32) {
	data[off] = byte(val)
	data[off+1] = byte(val >> 8)
	data[off+2] = byte(val >> 16)
	data[off+3] = byte(val >> 24)
}

func TestParseMinimalDex(t *testing.T) {
	data := buildMinimalDex()
	df, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse minimal DEX failed: %v", err)
	}

	if df.NumStringIDs() != 0 {
		t.Errorf("expected 0 strings, got %d", df.NumStringIDs())
	}
	if df.NumTypeIDs() != 0 {
		t.Errorf("expected 0 types, got %d", df.NumTypeIDs())
	}
	if df.NumMethodIDs() != 0 {
		t.Errorf("expected 0 methods, got %d", df.NumMethodIDs())
	}
}

func TestParseTooShort(t *testing.T) {
	_, err := Parse([]byte{0x01, 0x02})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestParseBadMagic(t *testing.T) {
	data := make([]byte, 112)
	copy(data[0:], "NOT_DEX!")
	_, err := Parse(data)
	if err == nil {
		t.Error("expected error for bad magic")
	}
}
