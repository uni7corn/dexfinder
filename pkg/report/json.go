package report

import (
	"encoding/json"
	"io"
	"strings"

	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
)

// JSONReport is the structured output format.
type JSONReport struct {
	Methods []JSONEntry `json:"methods,omitempty"`
	Fields  []JSONEntry `json:"fields,omitempty"`
	Summary JSONSummary `json:"summary"`
}

// JSONEntry represents a single API reference finding.
type JSONEntry struct {
	API     string       `json:"api"`
	Type    string       `json:"type"` // "method" or "field"
	Callers []JSONCaller `json:"callers"`
	Count   int          `json:"count"`
}

// JSONCaller represents where an API is called from.
type JSONCaller struct {
	Method     string `json:"method"`
	Occurrences int   `json:"occurrences"`
}

// JSONSummary provides overview statistics.
type JSONSummary struct {
	TotalMethods int `json:"total_methods"`
	TotalFields  int `json:"total_fields"`
	TotalClasses int `json:"total_classes"`
}

// DumpJSON writes scan results as JSON.
func DumpJSON(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string) error {
	report := JSONReport{
		Summary: JSONSummary{
			TotalMethods: len(result.MethodRefs),
			TotalFields:  len(result.FieldRefs),
			TotalClasses: len(result.Classes),
		},
	}

	for api, refs := range result.MethodRefs {
		if query != "" && !strings.Contains(strings.ToLower(api), strings.ToLower(query)) {
			continue
		}
		entry := JSONEntry{
			API:   api,
			Type:  "method",
			Count: len(refs),
		}
		callers := make(map[string]int)
		for _, ref := range refs {
			if ref.CallerDexIdx < len(dexFiles) {
				name := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
				callers[name]++
			}
		}
		for name, count := range callers {
			entry.Callers = append(entry.Callers, JSONCaller{Method: name, Occurrences: count})
		}
		report.Methods = append(report.Methods, entry)
	}

	for api, refs := range result.FieldRefs {
		if query != "" && !strings.Contains(strings.ToLower(api), strings.ToLower(query)) {
			continue
		}
		entry := JSONEntry{
			API:   api,
			Type:  "field",
			Count: len(refs),
		}
		callers := make(map[string]int)
		for _, ref := range refs {
			if ref.CallerDexIdx < len(dexFiles) {
				name := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
				callers[name]++
			}
		}
		for name, count := range callers {
			entry.Callers = append(entry.Callers, JSONCaller{Method: name, Occurrences: count})
		}
		report.Fields = append(report.Fields, entry)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
