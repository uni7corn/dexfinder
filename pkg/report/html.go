package report

import (
	"fmt"
	"html"
	"io"
	"strings"

	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/hiddenapi"
)

// DumpHTML writes scan results as a self-contained HTML report.
func DumpHTML(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string, scope finder.QueryScope, dc *DisplayConfig) {
	var opts []finder.QueryOption
	if dc != nil && dc.Mapping != nil {
		opts = append(opts, finder.QueryOption{Mapping: dc.Mapping})
	}
	qr := finder.Query(result, dexFiles, query, scope, opts...)

	htmlWriteHeader(w, "dexfinder — Scan Report", query)
	htmlWriteSummaryBar(w, len(qr.MatchedMethods), len(qr.MatchedFields), len(qr.MatchedStrings))

	// Methods
	apis := sortedKeys(qr.MatchedMethods)
	for _, api := range apis {
		refs := qr.MatchedMethods[api]
		callers := aggregateMethodCallers(refs, dexFiles, dc)
		htmlWriteEntry(w, "METHOD", dc.FormatAPI(api), callers, len(refs))
	}

	// Caller matches
	if query != "" {
		callerAPIs := sortedKeys(qr.MatchedCallers)
		for _, api := range callerAPIs {
			if _, shown := qr.MatchedMethods[api]; shown {
				continue
			}
			refs := qr.MatchedCallers[api]
			callers := aggregateMethodCallers(refs, dexFiles, dc)
			htmlWriteEntry(w, "CALLER→", dc.FormatAPI(api), callers, len(refs))
		}
	}

	// Fields
	fields := sortedFieldKeys(qr.MatchedFields)
	for _, api := range fields {
		refs := qr.MatchedFields[api]
		callers := aggregateFieldCallers(refs, dexFiles, dc)
		htmlWriteEntry(w, "FIELD", dc.FormatAPI(api), callers, len(refs))
	}

	// Strings
	if query != "" && len(qr.MatchedStrings) > 0 {
		strs := sortedStringKeys(qr.MatchedStrings)
		for _, str := range strs {
			refs := qr.MatchedStrings[str]
			callers := aggregateStringCallers(refs, dexFiles, dc)
			display := str
			if len(display) > 120 {
				display = display[:120] + "..."
			}
			htmlWriteEntry(w, "STRING", `"`+display+`"`, callers, len(refs))
		}
	}

	htmlWriteFooter(w)
}

// DumpTraceHTML writes trace results as an interactive HTML report.
func DumpTraceHTML(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string, maxDepth int, dc *DisplayConfig) {
	if query == "" {
		htmlWriteHeader(w, "dexfinder — Trace Report", "")
		fmt.Fprintf(w, `<p class="error">Error: --query is required for --trace mode</p>`)
		htmlWriteFooter(w)
		return
	}

	var opts []finder.QueryOption
	if dc != nil && dc.Mapping != nil {
		opts = append(opts, finder.QueryOption{Mapping: dc.Mapping})
	}
	qr := finder.Query(result, dexFiles, query, finder.ScopeCallee, opts...)

	htmlWriteHeader(w, "dexfinder — Trace Report", query)

	if len(qr.MatchedMethods) == 0 && len(qr.MatchedFields) == 0 {
		fmt.Fprintf(w, `<p class="no-results">No matching APIs found for: %s</p>`, html.EscapeString(query))
		htmlWriteFooter(w)
		return
	}

	cg := finder.BuildCallGraph(result, dexFiles)

	allAPIs := append(sortedKeys(qr.MatchedMethods), sortedFieldKeys(qr.MatchedFields)...)
	fmt.Fprintf(w, `<div class="summary-bar"><span>%d target API(s)</span> <span>depth: %d</span></div>`+"\n", len(allAPIs), maxDepth)

	for _, api := range allAPIs {
		tree := cg.TraceCallers(api, maxDepth)
		fmt.Fprintf(w, `<div class="trace-target">`+"\n")
		fmt.Fprintf(w, `<h3 class="target-api">%s</h3>`+"\n", html.EscapeString(dc.FormatHeader(api)))
		if len(tree.Callers) == 0 {
			fmt.Fprintf(w, `<p class="no-callers">(no callers found)</p>`+"\n")
		} else {
			fmt.Fprintf(w, `<ul class="call-tree">`+"\n")
			htmlWriteTreeNodes(w, tree, dc)
			fmt.Fprintf(w, `</ul>`+"\n")
		}
		fmt.Fprintf(w, `</div>`+"\n")
	}

	htmlWriteFooter(w)
}

// DumpHiddenAPIHTML writes hidden API findings as HTML.
func DumpHiddenAPIHTML(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, db *hiddenapi.Database, dc *DisplayConfig) {
	htmlWriteHeader(w, "dexfinder — Hidden API Report", "")

	stats := NewStats()

	// Linking: methods
	apis := sortedKeys(result.MethodRefs)
	for _, api := range apis {
		refs := result.MethodRefs[api]
		apiList := db.GetApiList(api)
		stats.LinkingCount++
		stats.ApiCounts[apiList]++
		stats.Count++
		callers := aggregateMethodCallers(refs, dexFiles, dc)
		htmlWriteHiddenEntry(w, stats.Count, "Linking", apiList.String(), api, callers)
	}

	// Linking: fields
	fields := sortedFieldKeys(result.FieldRefs)
	for _, api := range fields {
		refs := result.FieldRefs[api]
		apiList := db.GetApiList(api)
		stats.LinkingCount++
		stats.ApiCounts[apiList]++
		stats.Count++
		callers := aggregateFieldCallers(refs, dexFiles, dc)
		htmlWriteHiddenEntry(w, stats.Count, "Linking", apiList.String(), api, callers)
	}

	// Reflection
	reflections := result.FindPotentialReflection(db)
	for _, ref := range reflections {
		apiList := db.GetApiList(ref.Signature)
		stats.ReflectionCount++
		stats.ApiCounts[apiList]++
		stats.Count++
		callers := aggregateStringCallers(ref.StringRef, dexFiles, dc)
		htmlWriteHiddenEntry(w, stats.Count, "Reflection", apiList.String(), ref.Signature, callers)
	}

	fmt.Fprintf(w, `<div class="summary-bar final">`)
	fmt.Fprintf(w, `<strong>%d</strong> hidden API(s): <strong>%d</strong> linked, <strong>%d</strong> reflection`,
		stats.Count, stats.LinkingCount, stats.ReflectionCount)
	fmt.Fprintf(w, `</div>`+"\n")

	htmlWriteFooter(w)
}

// --- HTML building blocks ---

func htmlWriteHeader(w io.Writer, title, query string) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
:root { --bg: #1e1e2e; --fg: #cdd6f4; --surface: #313244; --border: #45475a; --blue: #89b4fa; --green: #a6e3a1; --red: #f38ba8; --yellow: #f9e2af; --cyan: #94e2d5; --mauve: #cba6f7; --dim: #6c7086; --overlay: #585b70; }
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace; font-size: 13px; background: var(--bg); color: var(--fg); padding: 20px; line-height: 1.6; }
h1 { font-size: 18px; color: var(--blue); margin-bottom: 4px; }
.subtitle { color: var(--dim); margin-bottom: 16px; font-size: 12px; }
.search-box { width: 100%%; padding: 8px 12px; background: var(--surface); color: var(--fg); border: 1px solid var(--border); border-radius: 6px; font-family: inherit; font-size: 13px; margin-bottom: 16px; outline: none; }
.search-box:focus { border-color: var(--blue); }
.summary-bar { background: var(--surface); padding: 8px 12px; border-radius: 6px; margin-bottom: 16px; color: var(--dim); display: flex; gap: 16px; flex-wrap: wrap; }
.summary-bar span, .summary-bar strong { color: var(--fg); }
.summary-bar.final { margin-top: 16px; border: 1px solid var(--border); }
.entry { background: var(--surface); border-radius: 6px; margin-bottom: 8px; overflow: hidden; border: 1px solid var(--border); }
.entry-header { padding: 8px 12px; cursor: pointer; display: flex; align-items: center; gap: 8px; user-select: none; }
.entry-header:hover { background: var(--overlay); }
.entry-header .arrow { transition: transform 0.15s; color: var(--dim); font-size: 10px; }
.entry.open .arrow { transform: rotate(90deg); }
.tag { font-weight: bold; padding: 1px 6px; border-radius: 3px; font-size: 11px; white-space: nowrap; }
.tag-METHOD { background: rgba(137,180,250,0.15); color: var(--blue); }
.tag-FIELD { background: rgba(148,226,213,0.15); color: var(--cyan); }
.tag-CALLER { background: rgba(166,227,161,0.15); color: var(--green); }
.tag-STRING { background: rgba(249,226,175,0.15); color: var(--yellow); }
.tag-STRING_TABLE { background: rgba(108,112,134,0.15); color: var(--dim); }
.tag-Linking { background: rgba(249,226,175,0.15); color: var(--yellow); }
.tag-Reflection { background: rgba(203,166,247,0.15); color: var(--mauve); }
.level-blocked { color: var(--red); font-weight: bold; }
.level-unsupported { color: var(--yellow); }
.api-name { flex: 1; word-break: break-all; }
.ref-count { color: var(--dim); white-space: nowrap; font-size: 12px; }
.entry-body { display: none; padding: 4px 12px 8px 32px; border-top: 1px solid var(--border); }
.entry.open .entry-body { display: block; }
.caller { color: var(--dim); padding: 2px 0; }
.caller .occ { color: var(--overlay); font-size: 11px; }
.no-results, .error { color: var(--red); padding: 20px; text-align: center; }
.no-callers { color: var(--dim); padding: 4px 0; font-style: italic; }
.call-tree, .call-tree ul { list-style: none; padding-left: 20px; }
.call-tree > li { padding-left: 0; }
.call-tree li { position: relative; padding: 2px 0; }
.call-tree li::before { content: ''; position: absolute; left: -14px; top: 0; bottom: 0; border-left: 1px solid var(--border); }
.call-tree li::after { content: ''; position: absolute; left: -14px; top: 12px; width: 10px; border-top: 1px solid var(--border); }
.call-tree li:last-child::before { height: 12px; }
.tree-toggle { cursor: pointer; user-select: none; }
.tree-toggle:hover { color: var(--blue); }
.cycle { color: var(--mauve); font-style: italic; }
.target-api { color: var(--blue); font-size: 14px; margin-bottom: 8px; padding: 8px 0; border-bottom: 1px solid var(--border); }
.trace-target { background: var(--surface); border-radius: 6px; padding: 12px 16px; margin-bottom: 12px; border: 1px solid var(--border); }
.hidden-entry { display: flex; align-items: baseline; gap: 8px; flex-wrap: wrap; }
.hidden-num { color: var(--dim); min-width: 30px; }
</style>
</head>
<body>
<h1>%s</h1>
`, html.EscapeString(title), html.EscapeString(title))
	if query != "" {
		fmt.Fprintf(w, `<div class="subtitle">Query: %s</div>`+"\n", html.EscapeString(query))
	}
	fmt.Fprintf(w, `<input type="text" class="search-box" placeholder="Filter results..." oninput="filterEntries(this.value)">`+"\n")
}

func htmlWriteFooter(w io.Writer) {
	fmt.Fprintf(w, `<script>
function filterEntries(q) {
  q = q.toLowerCase();
  document.querySelectorAll('.entry, .trace-target').forEach(function(el) {
    var text = el.textContent.toLowerCase();
    el.style.display = text.includes(q) ? '' : 'none';
  });
}
document.querySelectorAll('.entry-header').forEach(function(h) {
  h.addEventListener('click', function() {
    h.parentElement.classList.toggle('open');
  });
});
document.querySelectorAll('.tree-toggle').forEach(function(t) {
  t.addEventListener('click', function() {
    var ul = t.nextElementSibling;
    if (ul && ul.tagName === 'UL') {
      ul.style.display = ul.style.display === 'none' ? '' : 'none';
      t.textContent = ul.style.display === 'none'
        ? t.textContent.replace('▼', '▶')
        : t.textContent.replace('▶', '▼');
    }
  });
});
</script>
</body>
</html>
`)
}

func htmlWriteSummaryBar(w io.Writer, methods, fields, strings int) {
	fmt.Fprintf(w, `<div class="summary-bar">`)
	fmt.Fprintf(w, `<span><strong>%d</strong> methods</span>`, methods)
	fmt.Fprintf(w, `<span><strong>%d</strong> fields</span>`, fields)
	fmt.Fprintf(w, `<span><strong>%d</strong> strings</span>`, strings)
	fmt.Fprintf(w, `</div>`+"\n")
}

func htmlWriteEntry(w io.Writer, tag, api string, callers []callerInfo, refCount int) {
	tagClass := strings.ReplaceAll(tag, "→", "")
	fmt.Fprintf(w, `<div class="entry">`+"\n")
	fmt.Fprintf(w, `<div class="entry-header">`)
	fmt.Fprintf(w, `<span class="arrow">▶</span> `)
	fmt.Fprintf(w, `<span class="tag tag-%s">%s</span> `, html.EscapeString(tagClass), html.EscapeString(tag))
	fmt.Fprintf(w, `<span class="api-name">%s</span> `, html.EscapeString(api))
	fmt.Fprintf(w, `<span class="ref-count">(%d ref)</span>`, refCount)
	fmt.Fprintf(w, `</div>`+"\n")
	fmt.Fprintf(w, `<div class="entry-body">`+"\n")
	for _, c := range callers {
		if c.count > 1 {
			fmt.Fprintf(w, `<div class="caller">%s <span class="occ">(%d occurrences)</span></div>`+"\n",
				html.EscapeString(c.name), c.count)
		} else {
			fmt.Fprintf(w, `<div class="caller">%s</div>`+"\n", html.EscapeString(c.name))
		}
	}
	fmt.Fprintf(w, `</div></div>`+"\n")
}

func htmlWriteHiddenEntry(w io.Writer, num uint32, accessType, level, api string, callers []callerInfo) {
	levelClass := "level-" + level
	fmt.Fprintf(w, `<div class="entry">`+"\n")
	fmt.Fprintf(w, `<div class="entry-header">`)
	fmt.Fprintf(w, `<span class="arrow">▶</span> `)
	fmt.Fprintf(w, `<span class="hidden-num">#%d</span> `, num)
	fmt.Fprintf(w, `<span class="tag tag-%s">%s</span> `, html.EscapeString(accessType), html.EscapeString(accessType))
	fmt.Fprintf(w, `<span class="%s">%s</span> `, levelClass, html.EscapeString(level))
	fmt.Fprintf(w, `<span class="api-name">%s</span>`, html.EscapeString(api))
	fmt.Fprintf(w, `</div>`+"\n")
	fmt.Fprintf(w, `<div class="entry-body">`+"\n")
	for _, c := range callers {
		fmt.Fprintf(w, `<div class="caller">%s</div>`+"\n", html.EscapeString(c.name))
	}
	fmt.Fprintf(w, `</div></div>`+"\n")
}

func htmlWriteTreeNodes(w io.Writer, node *finder.CallChainNode, dc *DisplayConfig) {
	for _, caller := range node.Callers {
		label := html.EscapeString(dc.FormatNode(caller.Method))
		if caller.IsCycle {
			fmt.Fprintf(w, `<li><span class="cycle">%s ⟳ [recursive]</span></li>`+"\n", label)
			continue
		}
		if len(caller.Callers) > 0 {
			fmt.Fprintf(w, `<li><span class="tree-toggle">▼ %s</span>`+"\n", label)
			fmt.Fprintf(w, `<ul>`+"\n")
			htmlWriteTreeNodes(w, caller, dc)
			fmt.Fprintf(w, `</ul></li>`+"\n")
		} else {
			fmt.Fprintf(w, `<li>%s</li>`+"\n", label)
		}
	}
}

// --- Caller aggregation helpers ---

type callerInfo struct {
	name  string
	count int
}

func aggregateMethodCallers(refs []finder.MethodRef, dexFiles []*dex.DexFile, dc *DisplayConfig) []callerInfo {
	callers := make(map[string]int)
	for _, ref := range refs {
		if ref.CallerDexIdx < len(dexFiles) {
			raw := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
			callers[deobfName(raw, dc)]++
		}
	}
	return sortedCallerInfos(callers)
}

func aggregateFieldCallers(refs []finder.FieldRef, dexFiles []*dex.DexFile, dc *DisplayConfig) []callerInfo {
	callers := make(map[string]int)
	for _, ref := range refs {
		if ref.CallerDexIdx < len(dexFiles) {
			raw := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
			callers[deobfName(raw, dc)]++
		}
	}
	return sortedCallerInfos(callers)
}

func aggregateStringCallers(refs []finder.StringRef, dexFiles []*dex.DexFile, dc *DisplayConfig) []callerInfo {
	callers := make(map[string]int)
	for _, ref := range refs {
		if ref.CallerDexIdx < len(dexFiles) {
			raw := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
			callers[deobfName(raw, dc)]++
		}
	}
	return sortedCallerInfos(callers)
}

func sortedCallerInfos(m map[string]int) []callerInfo {
	keys := sortedCountKeys(m)
	infos := make([]callerInfo, 0, len(keys))
	for _, k := range keys {
		infos = append(infos, callerInfo{name: k, count: m[k]})
	}
	return infos
}
