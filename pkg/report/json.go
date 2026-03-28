package report

import (
	"encoding/json"
	"io"
	"strings"

	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
)

// --- Scan output (no trace) ---

type JSONReport struct {
	Methods []JSONEntry `json:"methods,omitempty"`
	Fields  []JSONEntry `json:"fields,omitempty"`
	Summary JSONSummary `json:"summary"`
}

type JSONEntry struct {
	API     string       `json:"api"`
	Type    string       `json:"type"`
	Callers []JSONCaller `json:"callers"`
	Count   int          `json:"count"`
}

type JSONCaller struct {
	Method      string `json:"method"`
	Occurrences int    `json:"occurrences"`
}

type JSONSummary struct {
	TotalMethods int `json:"total_methods"`
	TotalFields  int `json:"total_fields"`
	TotalClasses int `json:"total_classes"`
}

// DumpJSON writes scan results as JSON (no trace).
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
		entry := JSONEntry{API: api, Type: "method", Count: len(refs)}
		callers := make(map[string]int)
		for _, ref := range refs {
			if ref.CallerDexIdx < len(dexFiles) {
				callers[dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)]++
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
		entry := JSONEntry{API: api, Type: "field", Count: len(refs)}
		callers := make(map[string]int)
		for _, ref := range refs {
			if ref.CallerDexIdx < len(dexFiles) {
				callers[dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)]++
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

// --- Trace output (tree / list) ---

// JSONTraceReport is the top-level trace output.
type JSONTraceReport struct {
	Targets []JSONTraceTarget `json:"targets"`
}

// JSONTraceTarget is one target API with its call chains.
type JSONTraceTarget struct {
	API    string      `json:"api"`
	Tree   *JSONNode   `json:"tree,omitempty"`   // layout=tree
	Chains [][]string  `json:"chains,omitempty"` // layout=list
}

// JSONNode is a tree node for layout=tree.
type JSONNode struct {
	Method   string      `json:"method"`
	IsCycle  bool        `json:"is_cycle,omitempty"`
	Callers  []*JSONNode `json:"callers,omitempty"`
}

// DumpTraceJSON writes trace results as JSON.
func DumpTraceJSON(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string, maxDepth int, dc *DisplayConfig) error {
	var opts []finder.QueryOption
	if dc != nil && dc.Mapping != nil {
		opts = append(opts, finder.QueryOption{Mapping: dc.Mapping})
	}
	qr := finder.Query(result, dexFiles, query, finder.ScopeCallee, opts...)
	cg := finder.BuildCallGraph(result, dexFiles)

	report := JSONTraceReport{}

	allAPIs := append(sortedKeys(qr.MatchedMethods), sortedFieldKeys(qr.MatchedFields)...)
	for _, api := range allAPIs {
		tree := cg.TraceCallers(api, maxDepth)
		target := JSONTraceTarget{
			API: dc.FormatHeader(api),
		}

		if dc != nil && dc.Layout == LayoutList {
			chains := finder.FlatCallerChains(tree, 100)
			for _, chain := range chains {
				var formatted []string
				for j := len(chain) - 1; j >= 0; j-- {
					formatted = append(formatted, dc.FormatNode(chain[j]))
				}
				target.Chains = append(target.Chains, formatted)
			}
		} else {
			target.Tree = buildJSONTree(tree, dc)
		}

		report.Targets = append(report.Targets, target)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func buildJSONTree(node *finder.CallChainNode, dc *DisplayConfig) *JSONNode {
	jn := &JSONNode{
		Method:  dc.FormatNode(node.Method),
		IsCycle: node.IsCycle,
	}
	for _, caller := range node.Callers {
		jn.Callers = append(jn.Callers, buildJSONTree(caller, dc))
	}
	return jn
}
