package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/hiddenapi"
)

// DumpScan outputs scan results in text format (scan mode, no CSV needed).
func DumpScan(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string, scope finder.QueryScope, dc *DisplayConfig) {
	var opts []finder.QueryOption
	if dc != nil && dc.Mapping != nil {
		opts = append(opts, finder.QueryOption{Mapping: dc.Mapping})
	}
	qr := finder.Query(result, dexFiles, query, scope, opts...)

	col := dc.Color

	// Matched methods (by callee API name)
	apis := sortedKeys(qr.MatchedMethods)
	for _, api := range apis {
		refs := qr.MatchedMethods[api]
		tag := colorTag(col, "[METHOD]")
		fmt.Fprintf(w, "%s %s %s\n", tag, dc.FormatAPI(api), colorCount(col, len(refs), "ref"))
		printMethodCallersDeobf(w, refs, dexFiles, dc)
	}

	// Matched methods (by caller name) — only if there's a query and caller scope
	if query != "" {
		callerAPIs := sortedKeys(qr.MatchedCallers)
		for _, api := range callerAPIs {
			if _, shown := qr.MatchedMethods[api]; shown {
				continue
			}
			refs := qr.MatchedCallers[api]
			tag := colorTag(col, "[CALLER→]")
			fmt.Fprintf(w, "%s %s %s\n", tag, dc.FormatAPI(api), colorCount(col, len(refs), "ref from matching callers"))
			printMethodCallersDeobf(w, refs, dexFiles, dc)
		}
	}

	// Matched fields
	fields := sortedFieldKeys(qr.MatchedFields)
	for _, api := range fields {
		refs := qr.MatchedFields[api]
		tag := colorTag(col, "[FIELD]")
		fmt.Fprintf(w, "%s  %s %s\n", tag, dc.FormatAPI(api), colorCount(col, len(refs), "ref"))
		printFieldCallersDeobf(w, refs, dexFiles, dc)
	}

	// Matched strings from code
	if query != "" && len(qr.MatchedStrings) > 0 {
		strs := sortedStringKeys(qr.MatchedStrings)
		for _, str := range strs {
			refs := qr.MatchedStrings[str]
			display := str
			if len(display) > 120 {
				display = display[:120] + "..."
			}
			tag := colorTag(col, "[STRING]")
			fmt.Fprintf(w, "%s \"%s\" %s\n", tag, display, colorCount(col, len(refs), "ref"))
			printStringCallersDeobf(w, refs, dexFiles, dc)
		}
	}

	// Matched strings from full string table (no caller info — these may be in annotations, static fields, etc.)
	if query != "" && len(qr.MatchedStringTable) > 0 {
		sort.Strings(qr.MatchedStringTable)
		for _, str := range qr.MatchedStringTable {
			display := str
			if len(display) > 120 {
				display = display[:120] + "..."
			}
			tag := colorTag(col, "[STRING_TABLE]")
			fmt.Fprintf(w, "%s \"%s\" (in DEX string table, no code reference found)\n", tag, display)
		}
	}
}

// DumpTrace outputs call chain trace for matched APIs.
func DumpTrace(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string, maxDepth int, dc *DisplayConfig) {
	if query == "" {
		fmt.Fprintln(w, "Error: --query is required for --trace mode")
		return
	}

	var opts []finder.QueryOption
	if dc != nil && dc.Mapping != nil {
		opts = append(opts, finder.QueryOption{Mapping: dc.Mapping})
	}
	qr := finder.Query(result, dexFiles, query, finder.ScopeCallee, opts...)

	if len(qr.MatchedMethods) == 0 && len(qr.MatchedFields) == 0 {
		fmt.Fprintf(w, "No matching APIs found for: %s\n", query)
		return
	}

	cg := finder.BuildCallGraph(result, dexFiles)

	layout := LayoutTree
	if dc != nil && dc.Layout == LayoutList {
		layout = LayoutList
	}

	if layout == LayoutList {
		dumpTraceList(w, qr, cg, maxDepth, dc)
	} else {
		dumpTraceTree(w, qr, cg, maxDepth, dc)
	}
}

// dumpTraceTree renders call chains as a merged tree.
func dumpTraceTree(w io.Writer, qr *finder.QueryResult, cg *finder.CallGraph, maxDepth int, dc *DisplayConfig) {
	allAPIs := append(sortedKeys(qr.MatchedMethods), sortedFieldKeys(qr.MatchedFields)...)
	for _, api := range allAPIs {
		fmt.Fprintf(w, "%s\n", dc.FormatHeader(api))
		tree := cg.TraceCallers(api, maxDepth)
		if len(tree.Callers) == 0 {
			fmt.Fprintln(w, "  (no callers found)")
		} else {
			printTreeNodes(w, tree, "", dc)
		}
		fmt.Fprintln(w)
	}
}

func printTreeNodes(w io.Writer, node *finder.CallChainNode, prefix string, dc *DisplayConfig) {
	col := dc.Color
	for i, caller := range node.Callers {
		isLast := i == len(node.Callers)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		label := dc.FormatNode(caller.Method)
		if caller.IsCycle {
			cycleMarker := " ⟳ [recursive]"
			if col.Enabled() {
				cycleMarker = " " + col.Cycle("⟳ [recursive]")
			}
			label += cycleMarker
		}
		fmt.Fprintf(w, "%s%s%s\n", colorTreePrefix(col, prefix), colorTreeConnector(col, connector), label)
		if !caller.IsCycle {
			childPrefix := prefix + "│   "
			if isLast {
				childPrefix = prefix + "    "
			}
			printTreeNodes(w, caller, childPrefix, dc)
		}
	}
}

// dumpTraceList renders call chains as flat individual chains.
func dumpTraceList(w io.Writer, qr *finder.QueryResult, cg *finder.CallGraph, maxDepth int, dc *DisplayConfig) {
	col := dc.Color
	allAPIs := append(sortedKeys(qr.MatchedMethods), sortedFieldKeys(qr.MatchedFields)...)
	for _, api := range allAPIs {
		tree := cg.TraceCallers(api, maxDepth)
		chains := finder.FlatCallerChains(tree, 100)

		if len(chains) == 0 {
			header := fmt.Sprintf("--- %s ---", dc.FormatHeader(api))
			fmt.Fprintln(w, colorChainHeader(col, header))
			fmt.Fprintln(w, "    (no callers found)")
			fmt.Fprintln(w)
			continue
		}

		for i, chain := range chains {
			header := fmt.Sprintf("--- Call chain #%d for %s ---", i+1, dc.FormatHeader(api))
			fmt.Fprintln(w, colorChainHeader(col, header))
			for j := len(chain) - 1; j >= 0; j-- {
				fmt.Fprintf(w, "\tat %s\n", dc.FormatNode(chain[j]))
			}
			fmt.Fprintln(w)
		}
	}
}

// shortName extracts a shorter display name from a full DEX signature.
// "Lcom/example/Foo;->bar(I)V" → "Foo.bar(I)V"
func shortName(fullAPI string) string {
	// Find class and method parts
	arrowIdx := strings.Index(fullAPI, "->")
	if arrowIdx == -1 {
		return fullAPI
	}
	classDesc := fullAPI[:arrowIdx]
	member := fullAPI[arrowIdx+2:]

	// "Lcom/example/Foo;" → "Foo"
	className := classDesc
	if strings.HasPrefix(className, "L") && strings.HasSuffix(className, ";") {
		className = className[1 : len(className)-1]
	}
	if lastSlash := strings.LastIndex(className, "/"); lastSlash != -1 {
		className = className[lastSlash+1:]
	}
	// Handle inner classes: "Foo$Bar" stays
	return className + "." + member
}

// DumpHiddenAPI outputs hidden API findings in veridex-compatible text format.
func DumpHiddenAPI(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, db *hiddenapi.Database, dc *DisplayConfig) *Stats {
	stats := NewStats()
	var col *Colorizer
	if dc != nil {
		col = dc.Color
	}

	// Direct linking: method references
	apis := sortedKeys(result.MethodRefs)
	for _, api := range apis {
		refs := result.MethodRefs[api]
		apiList := db.GetApiList(api)
		stats.LinkingCount++
		stats.ApiCounts[apiList]++
		stats.Count++
		level := colorHiddenLevel(col, apiList.String())
		fmt.Fprintf(w, "#%d: Linking %s %s use(s):\n", stats.Count, level, api)
		printMethodCallers(w, refs, dexFiles)
		fmt.Fprintln(w)
	}

	// Direct linking: field references
	fields := sortedFieldKeys(result.FieldRefs)
	for _, api := range fields {
		refs := result.FieldRefs[api]
		apiList := db.GetApiList(api)
		stats.LinkingCount++
		stats.ApiCounts[apiList]++
		stats.Count++
		level := colorHiddenLevel(col, apiList.String())
		fmt.Fprintf(w, "#%d: Linking %s %s use(s):\n", stats.Count, level, api)
		printFieldCallers(w, refs, dexFiles)
		fmt.Fprintln(w)
	}

	// Imprecise reflection: class × string cross-matching (veridex-compatible)
	reflections := result.FindPotentialReflection(db)
	for _, ref := range reflections {
		apiList := db.GetApiList(ref.Signature)
		stats.ReflectionCount++
		stats.ApiCounts[apiList]++
		stats.Count++
		level := colorHiddenLevel(col, apiList.String())
		fmt.Fprintf(w, "#%d: Reflection %s %s potential use(s):\n", stats.Count, level, ref.Signature)
		printStringCallers(w, ref.StringRef, dexFiles)
		fmt.Fprintln(w)
	}

	summary := fmt.Sprintf("%d hidden API(s) used: %d linked against, %d through reflection",
		stats.Count, stats.LinkingCount, stats.ReflectionCount)
	fmt.Fprintln(w, colorSummary(col, summary))

	return stats
}

func deobfName(dexSig string, dc *DisplayConfig) string {
	if dc == nil || dc.Mapping == nil {
		return dexSig
	}
	return dc.FormatAPI(dexSig)
}

func printMethodCallersDeobf(w io.Writer, refs []finder.MethodRef, dexFiles []*dex.DexFile, dc *DisplayConfig) {
	callers := make(map[string]int)
	for _, ref := range refs {
		if ref.CallerDexIdx < len(dexFiles) {
			raw := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
			callers[deobfName(raw, dc)]++
		}
	}
	sorted := sortedCountKeys(callers)
	for _, name := range sorted {
		count := callers[name]
		if count > 1 {
			fmt.Fprintf(w, "       %s (%d occurrences)\n", name, count)
		} else {
			fmt.Fprintf(w, "       %s\n", name)
		}
	}
}

func printFieldCallersDeobf(w io.Writer, refs []finder.FieldRef, dexFiles []*dex.DexFile, dc *DisplayConfig) {
	callers := make(map[string]int)
	for _, ref := range refs {
		if ref.CallerDexIdx < len(dexFiles) {
			raw := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
			callers[deobfName(raw, dc)]++
		}
	}
	sorted := sortedCountKeys(callers)
	for _, name := range sorted {
		count := callers[name]
		if count > 1 {
			fmt.Fprintf(w, "       %s (%d occurrences)\n", name, count)
		} else {
			fmt.Fprintf(w, "       %s\n", name)
		}
	}
}

func printStringCallersDeobf(w io.Writer, refs []finder.StringRef, dexFiles []*dex.DexFile, dc *DisplayConfig) {
	callers := make(map[string]int)
	for _, ref := range refs {
		if ref.CallerDexIdx < len(dexFiles) {
			raw := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
			callers[deobfName(raw, dc)]++
		}
	}
	sorted := sortedCountKeys(callers)
	for _, name := range sorted {
		count := callers[name]
		if count > 1 {
			fmt.Fprintf(w, "       %s (%d occurrences)\n", name, count)
		} else {
			fmt.Fprintf(w, "       %s\n", name)
		}
	}
}

// Keep old versions for DumpHiddenAPI (doesn't use DisplayConfig)
func printMethodCallers(w io.Writer, refs []finder.MethodRef, dexFiles []*dex.DexFile) {
	printMethodCallersDeobf(w, refs, dexFiles, nil)
}

func printFieldCallers(w io.Writer, refs []finder.FieldRef, dexFiles []*dex.DexFile) {
	printFieldCallersDeobf(w, refs, dexFiles, nil)
}

func printStringCallers(w io.Writer, refs []finder.StringRef, dexFiles []*dex.DexFile) {
	printStringCallersDeobf(w, refs, dexFiles, nil)
}

func sortedKeys(m map[string][]finder.MethodRef) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedFieldKeys(m map[string][]finder.FieldRef) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringKeys(m map[string][]finder.StringRef) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedCountKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- Color helpers (nil-safe) ---

func colorTag(c *Colorizer, tag string) string {
	if c == nil {
		return tag
	}
	return c.Tag(tag)
}

func colorCount(c *Colorizer, n int, unit string) string {
	if c == nil {
		return fmt.Sprintf("(%d %s)", n, unit)
	}
	return c.Count(n, unit)
}

func colorTreeConnector(c *Colorizer, s string) string {
	if c == nil {
		return s
	}
	return c.TreeConnector(s)
}

func colorTreePrefix(c *Colorizer, s string) string {
	if c == nil {
		return s
	}
	return c.TreeConnector(s)
}

func colorChainHeader(c *Colorizer, s string) string {
	if c == nil {
		return s
	}
	return c.ChainHeader(s)
}

func colorHiddenLevel(c *Colorizer, s string) string {
	if c == nil {
		return s
	}
	return c.HiddenAPILevel(s)
}

func colorSummary(c *Colorizer, s string) string {
	if c == nil {
		return s
	}
	return c.Summary(s)
}
