package apk

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dex_method_finder/pkg/dex"
)

// LoadDexFiles loads all DEX files from an APK or a single DEX file.
// Supports: .apk (ZIP containing classes*.dex), .dex (raw DEX), .jar/.aar (ZIP).
// All data is loaded into memory, no temp files.
func LoadDexFiles(path string) ([]*dex.DexFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	// Check if it's a ZIP (APK/JAR/AAR)
	if ext == ".apk" || ext == ".jar" || ext == ".aar" || isZip(data) {
		return loadFromZip(data, path)
	}

	// Try as raw DEX
	if isDex(data) {
		df, err := dex.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("parse DEX %s: %w", path, err)
		}
		return []*dex.DexFile{df}, nil
	}

	return nil, fmt.Errorf("unrecognized file format: %s", path)
}

// loadFromZip extracts and parses all classes*.dex from a ZIP archive in memory.
func loadFromZip(data []byte, path string) ([]*dex.DexFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open ZIP %s: %w", path, err)
	}

	// Collect DEX entries, sorted by name for deterministic order
	var dexEntries []*zip.File
	for _, f := range reader.File {
		name := filepath.Base(f.Name)
		if isDexFileName(name) {
			dexEntries = append(dexEntries, f)
		}
	}

	sort.Slice(dexEntries, func(i, j int) bool {
		return dexEntries[i].Name < dexEntries[j].Name
	})

	if len(dexEntries) == 0 {
		return nil, fmt.Errorf("no DEX files found in %s", path)
	}

	var dexFiles []*dex.DexFile
	for _, entry := range dexEntries {
		rc, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s in ZIP: %w", entry.Name, err)
		}

		dexData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s in ZIP: %w", entry.Name, err)
		}

		df, err := dex.Parse(dexData)
		if err != nil {
			return nil, fmt.Errorf("parse %s in ZIP: %w", entry.Name, err)
		}
		dexFiles = append(dexFiles, df)
	}

	return dexFiles, nil
}

func isDexFileName(name string) bool {
	// classes.dex, classes2.dex, classes3.dex, ...
	if name == "classes.dex" {
		return true
	}
	if strings.HasPrefix(name, "classes") && strings.HasSuffix(name, ".dex") {
		return true
	}
	return false
}

func isZip(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

func isDex(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == "dex\n"
}
