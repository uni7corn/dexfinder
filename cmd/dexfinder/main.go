package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"dex_method_finder/pkg/apk"
	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/hiddenapi"
	"dex_method_finder/pkg/mapping"
	"dex_method_finder/pkg/model"
	"dex_method_finder/pkg/report"
)

var version = "dev"

var (
	flagDexFile     = flag.String("dex-file", "", "APK/DEX/JAR file to analyze (required)")
	flagQuery       = flag.String("query", "", "Search keyword (Java-style, DEX/JNI-style, or simple name)")
	flagFormat      = flag.String("format", "text", "Output format: text, json, model")
	flagLayout      = flag.String("layout", "tree", "Trace layout: tree (merged paths) or list (flat chains)")
	flagStyle       = flag.String("style", "java", "Name style: java (readable) or dex (JNI signatures)")
	flagFlagsFile   = flag.String("api-flags", "", "Path to hiddenapi-flags.csv (enables hidden API detection)")
	flagClassFilter = flag.String("class-filter", "", "Comma-separated class descriptor prefixes to include")
	flagExclude     = flag.String("exclude-api-lists", "", "Comma-separated API lists to exclude from reporting")
	flagShowStats   = flag.Bool("stats", false, "Show summary statistics only")
	flagTrace       = flag.Bool("trace", false, "Trace call chains for matched APIs (requires --query)")
	flagDepth       = flag.Int("depth", 5, "Max call chain depth for --trace")
	flagMapping     = flag.String("mapping", "", "ProGuard/R8 mapping.txt for deobfuscation")
	flagShowObf     = flag.Bool("show-obf", false, "Show obfuscated names alongside deobfuscated (requires --mapping)")
	flagScope       = flag.String("scope", "all", "Search scope: all, callee, caller, string, string-table, everything")
	flagVersion     = flag.Bool("version", false, "Show version")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `dexfinder %s — Cross-platform APK/DEX method & field reference finder

USAGE:
  dexfinder --dex-file <path> [options]

  The tool auto-selects mode based on flags:
    (default)        Scan and list references matching --query
    --trace          Trace call chains (requires --query)
    --api-flags      Detect hidden API usage (linking + reflection)

QUERY (--query):
  Accepts multiple formats, auto-detected:
    getDeviceId                                          simple name (fuzzy)
    android.telephony.TelephonyManager                   Java class (all methods)
    android.telephony.TelephonyManager#getDeviceId       Java class#method (all overloads)
    ...TelephonyManager#getDeviceId()                    Java with params (exact + fallback)
    Landroid/telephony/TelephonyManager;->getDeviceId()Ljava/lang/String;   DEX/JNI (exact)

OUTPUT (--format):
    text     Plain text with [METHOD]/[FIELD]/[STRING] tags (default)
    json     JSON output; with --trace supports tree/list layout
    model    Structured JSON with MethodInfo/FieldInfo types (for IDE/CI)

TRACE (--trace, requires --query):
    --layout tree    Merged tree — shared paths collapsed (default)
    --layout list    Flat list — each call chain shown independently
    --style  java    Java names: com.foo.Bar.method(Bar.java) (default)
    --style  dex     DEX signatures: Bar.method(Ljava/lang/String;)V
    --depth  N       Max call chain depth (default 5)
  Note: --layout and --style only affect --trace output.

SCOPE (--scope):
    all              Who calls this API? + string matches (default)
    callee           Who calls this API? (method/field references only)
    caller           What does this method call internally? (reverse direction)
    string           String constants in code (const-string instructions)
    string-table     Code strings + full DEX string table (annotations, dead code)
    everything       All of the above

DEOBFUSCATION (--mapping):
    --mapping mapping.txt    Load ProGuard/R8 mapping for name deobfuscation
    --show-obf               Show both original and obfuscated names (requires --mapping)

HIDDEN API (--api-flags):
    --api-flags hiddenapi-flags.csv    Detect blocked/unsupported API usage
    --exclude-api-lists sdk,unsupported    Skip these restriction levels in output
      Valid levels: sdk, unsupported, blocked, max-target-o/p/q/r/s

CLASS FILTER (--class-filter):
    Comma-separated DEX class descriptor prefixes. Only scan classes matching these.
    Format: use L prefix and / separator.
    Example: --class-filter "Lcom/mycompany/,Lcom/mylib/"

EXAMPLES:
  dexfinder --dex-file app.apk --stats
  dexfinder --dex-file app.apk --query "getDeviceId"
  dexfinder --dex-file app.apk --query "getDeviceId" --trace --depth 8
  dexfinder --dex-file app.apk --query "getDeviceId" --trace --layout list
  dexfinder --dex-file app.apk --query "getDeviceId" --trace --format json
  dexfinder --dex-file app.apk --query "getDeviceId" --trace --mapping mapping.txt --show-obf
  dexfinder --dex-file app.apk --query "getDeviceId" --scope caller
  dexfinder --dex-file app.apk --query "content://contacts" --scope string-table
  dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv
  dexfinder --dex-file app.apk --api-flags hiddenapi-flags.csv --exclude-api-lists sdk
  dexfinder --dex-file app.apk --query "getDeviceId" --class-filter "Lcom/mycompany/"

OPTIONS:
`, version)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flagVersion {
		fmt.Printf("dexfinder %s\n", version)
		return
	}

	if *flagDexFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	start := time.Now()

	// Load DEX files
	fmt.Fprintf(os.Stderr, "Loading %s ...\n", *flagDexFile)
	dexFiles, err := apk.LoadDexFiles(*flagDexFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	totalMethods := uint32(0)
	totalClasses := uint32(0)
	for _, df := range dexFiles {
		totalMethods += df.NumMethodIDs()
		totalClasses += df.Header.ClassDefsSize
	}
	fmt.Fprintf(os.Stderr, "Loaded %d DEX file(s): %d classes, %d method refs\n",
		len(dexFiles), totalClasses, totalMethods)

	// Load mapping (optional)
	var pm *mapping.ProguardMapping
	if *flagMapping != "" {
		fmt.Fprintf(os.Stderr, "Loading mapping from %s ...\n", *flagMapping)
		pm, err = mapping.LoadProguardMapping(*flagMapping)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading mapping: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Loaded %d class mappings\n", pm.Size())
	}

	style := report.StyleJava
	if *flagStyle == "dex" {
		style = report.StyleDex
	}
	layout := report.LayoutTree
	if *flagLayout == "list" {
		layout = report.LayoutList
	}

	dc := &report.DisplayConfig{
		Mapping: pm,
		ShowObf: *flagShowObf,
		Format:  report.OutputFormat(*flagFormat),
		Layout:  layout,
		Style:   style,
	}

	// Build class filter
	var prefixes []string
	if *flagClassFilter != "" {
		prefixes = strings.Split(*flagClassFilter, ",")
	}
	classFilter := finder.NewClassFilter(prefixes)

	// Load hidden API database (optional)
	var db *hiddenapi.Database
	if *flagFlagsFile != "" {
		var excludeNames []string
		if *flagExclude != "" {
			excludeNames = strings.Split(*flagExclude, ",")
		}
		filter := hiddenapi.NewApiListFilter(excludeNames)
		db = hiddenapi.NewDatabase(filter)
		fmt.Fprintf(os.Stderr, "Loading API flags from %s ...\n", *flagFlagsFile)
		if err := db.LoadFromFile(*flagFlagsFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading flags: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Loaded %d API entries\n", db.Size())
	}

	// Parse scope
	scope := parseScope(*flagScope)

	// Scan
	fmt.Fprintf(os.Stderr, "Scanning ...\n")
	directFinder := finder.NewDirectFinder(dexFiles, classFilter, db)
	result := directFinder.Scan()

	elapsed := time.Since(start)

	// Stats mode
	if *flagShowStats {
		fmt.Printf("Method references: %d\n", len(result.MethodRefs))
		fmt.Printf("Field references:  %d\n", len(result.FieldRefs))
		fmt.Printf("String constants:  %d\n", len(result.StringRefs))
		fmt.Printf("Referenced types:  %d\n", len(result.Classes))
		fmt.Printf("Time: %v\n", elapsed)
		return
	}

	// Use buffered writer for large output
	bw := bufio.NewWriterSize(os.Stdout, 256*1024)
	defer bw.Flush()

	// Structured model output
	if *flagFormat == "model" {
		outputModel(bw, result, dexFiles, pm, db, scope, start)
		return
	}

	// Trace mode
	if *flagTrace && *flagFormat != "model" {
		fmt.Fprintf(os.Stderr, "Building call graph ...\n")
		if *flagFormat == "json" {
			report.DumpTraceJSON(bw, result, dexFiles, *flagQuery, *flagDepth, dc)
		} else {
			report.DumpTrace(bw, result, dexFiles, *flagQuery, *flagDepth, dc)
		}
		fmt.Fprintf(os.Stderr, "Done in %v\n", time.Since(start))
		return
	}

	// Hidden API mode
	if db != nil {
		filtered := result.FilterHiddenAPIs(db)
		if *flagFormat == "json" {
			report.DumpJSON(bw, filtered, dexFiles, *flagQuery)
		} else {
			report.DumpHiddenAPI(bw, filtered, dexFiles, db)
		}
	} else {
		// Scan mode
		if *flagFormat == "json" {
			report.DumpJSON(bw, result, dexFiles, *flagQuery)
		} else {
			report.DumpScan(bw, result, dexFiles, *flagQuery, scope)
		}
	}

	fmt.Fprintf(os.Stderr, "Done in %v\n", elapsed)
}

func outputModel(bw *bufio.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, pm *mapping.ProguardMapping, db *hiddenapi.Database, scope finder.QueryScope, start time.Time) {
	conv := &model.Converter{DexFiles: dexFiles, Mapping: pm}
	meta := model.Metadata{
		FilePath:    *flagDexFile,
		DexCount:    len(dexFiles),
		Query:       *flagQuery,
		MappingFile: *flagMapping,
		FlagsFile:   *flagFlagsFile,
	}
	qr := finder.Query(result, dexFiles, *flagQuery, scope)
	filteredResult := &finder.ScanResult{
		MethodRefs: qr.MatchedMethods,
		FieldRefs:  qr.MatchedFields,
		StringRefs: qr.MatchedStrings,
		Classes:    result.Classes,
		AllStrings: result.AllStrings,
	}
	ar := conv.ConvertScanResult(filteredResult, meta)

	if *flagTrace && *flagQuery != "" {
		cg := finder.BuildCallGraph(result, dexFiles)
		for api := range qr.MatchedMethods {
			tree := cg.TraceCallers(api, *flagDepth)
			ar.CallChains = append(ar.CallChains, conv.ConvertCallChains(tree)...)
		}
		ar.Summary.CallChainsCount = len(ar.CallChains)
	}

	if db != nil {
		filtered := result.FilterHiddenAPIs(db)
		for api := range filtered.MethodRefs {
			ar.HiddenAPIs = append(ar.HiddenAPIs, model.HiddenAPIFinding{
				Signature:   api,
				Restriction: db.GetApiList(api).String(),
				AccessType:  "linking",
			})
		}
		reflections := result.FindPotentialReflection(db)
		for _, ref := range reflections {
			ar.ReflectionFindings = append(ar.ReflectionFindings, model.ReflectionFinding{
				Signature:   ref.Signature,
				TargetClass: ref.Class,
				MemberName:  ref.Member,
				Restriction: db.GetApiList(ref.Signature).String(),
			})
		}
		ar.Summary.HiddenAPICount = len(ar.HiddenAPIs) + len(ar.ReflectionFindings)
		ar.Summary.LinkingCount = len(ar.HiddenAPIs)
		ar.Summary.ReflectionCount = len(ar.ReflectionFindings)
	}

	enc := json.NewEncoder(bw)
	enc.SetIndent("", "  ")
	enc.Encode(ar)
	fmt.Fprintf(os.Stderr, "Done in %v\n", time.Since(start))
}

func parseScope(s string) finder.QueryScope {
	switch s {
	case "callee":
		return finder.ScopeCallee
	case "caller":
		return finder.ScopeCaller
	case "string":
		return finder.ScopeString
	case "string-table":
		return finder.ScopeString | finder.ScopeStringTable
	case "everything":
		return finder.ScopeEverything
	default:
		return finder.ScopeAll
	}
}
