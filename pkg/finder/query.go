package finder

import (
	"strings"

	"dex_method_finder/pkg/dex"
)

// QueryScope defines what to search in.
type QueryScope uint8

const (
	ScopeCallee      QueryScope = 1 << iota // match target API (被调用方)
	ScopeCaller                             // match caller method (调用方)
	ScopeString                             // match string constants in code (const-string)
	ScopeStringTable                        // match full DEX string table (annotations, debug, etc.)
	ScopeAll         = ScopeCallee | ScopeString                   // default: callee + strings (no caller)
	ScopeEverything  = ScopeCallee | ScopeCaller | ScopeString | ScopeStringTable
)

// QueryResult holds matched items from a query.
type QueryResult struct {
	MatchedMethods      map[string][]MethodRef  // callee API matches
	MatchedCallers      map[string][]MethodRef  // caller method matches
	MatchedFields       map[string][]FieldRef   // field API matches
	MatchedStrings      map[string][]StringRef  // string constant matches (from code)
	MatchedStringTable  []string                // matches from full DEX string table (no caller info)
}

// queryMatcher handles multiple input formats for matching.
type queryMatcher struct {
	// All normalized forms to try matching against
	patterns []string
	exact    bool // if true, match full signature exactly (not substring)
}

// newQueryMatcher builds a matcher from the user's query string.
// Supports:
//
//  1. DEX/JNI 签名:  Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V
//  2. Java 全限定名:  android.location.LocationManager#requestLocationUpdates(java.lang.String, long, float, android.location.LocationListener)
//  3. Java 类名:     android.location.LocationManager
//  4. 简单方法名:     requestLocationUpdates
//  5. 简单类名:       LocationManager
//  6. 部分路径:       location/LocationManager
func newQueryMatcher(query string) *queryMatcher {
	m := &queryMatcher{}

	query = strings.TrimSpace(query)
	if query == "" {
		return m
	}

	// Always add the raw query
	m.patterns = append(m.patterns, query)

	// Detect if it's already a DEX-style signature (contains L...;-> or ->)
	if strings.Contains(query, "->") || (strings.HasPrefix(query, "L") && strings.Contains(query, ";")) {
		// It's DEX format, also generate Java-style for cross-matching
		javaForm := dexToJavaStyle(query)
		if javaForm != query {
			m.patterns = append(m.patterns, javaForm)
		}
		return m
	}

	// Detect Java-style: has dots and possibly # or (
	if strings.Contains(query, ".") {
		// Java class/method name → convert to DEX format
		dexForm := javaToDexStyle(query)
		if dexForm != query {
			m.patterns = append(m.patterns, dexForm)
		}

		// Also try with just slash conversion (no L prefix) for partial matching
		slashForm := strings.ReplaceAll(query, ".", "/")
		if slashForm != query {
			m.patterns = append(m.patterns, slashForm)
		}

		// Also generate the full DEX class descriptor
		// "android.location.LocationManager" → "Landroid/location/LocationManager;"
		parts := strings.SplitN(query, "#", 2)
		classSlash := strings.ReplaceAll(parts[0], ".", "/")
		classDesc := "L" + classSlash + ";"
		m.patterns = append(m.patterns, classDesc)

		if len(parts) == 2 {
			methodPart := parts[1]
			if paren := strings.Index(methodPart, "("); paren != -1 {
				// User provided full signature with params → precise mode
				// Only keep the exact DEX signature, drop broad class patterns
				dexSig := classDesc + "->" + methodPart[:paren] + javaParamsToDex(methodPart[paren:])
				m.patterns = []string{dexSig}
				// Also add class->method (no params) for fuzzy fallback
				m.patterns = append(m.patterns, classDesc+"->"+methodPart[:paren])
				return m
			}
			m.patterns = append(m.patterns, classDesc+"->"+methodPart)
		}
	}

	// If it contains '/' but no 'L' prefix, add L...;
	if strings.Contains(query, "/") && !strings.HasPrefix(query, "L") {
		parts := strings.SplitN(query, "->", 2)
		classDesc := "L" + parts[0] + ";"
		m.patterns = append(m.patterns, classDesc)
		if len(parts) == 2 {
			m.patterns = append(m.patterns, classDesc+"->"+parts[1])
		}
	}

	// Deduplicate patterns
	seen := make(map[string]bool)
	var unique []string
	for _, p := range m.patterns {
		lp := strings.ToLower(p)
		if !seen[lp] {
			seen[lp] = true
			unique = append(unique, p)
		}
	}
	m.patterns = unique

	return m
}

// matches returns true if any pattern matches the target string.
// If an exact DEX signature pattern exists (contains "->" and "("), use exact match for that.
// Otherwise use case-insensitive substring match.
func (m *queryMatcher) matches(target string) bool {
	if len(m.patterns) == 0 {
		return true
	}
	targetLower := strings.ToLower(target)
	for _, p := range m.patterns {
		pLower := strings.ToLower(p)
		// If pattern looks like a full DEX signature with params → exact match
		if strings.Contains(p, "->") && strings.Contains(p, "(") && strings.Contains(p, ")") {
			if targetLower == pLower {
				return true
			}
			continue
		}
		// Otherwise substring match
		if strings.Contains(targetLower, pLower) {
			return true
		}
	}
	return false
}

// javaToDexStyle converts a Java-style method reference to DEX-style.
// "android.location.LocationManager#requestLocationUpdates(java.lang.String, long, float, android.location.LocationListener)"
// → "Landroid/location/LocationManager;->requestLocationUpdates(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
func javaToDexStyle(query string) string {
	// Split class#method
	parts := strings.SplitN(query, "#", 2)
	className := strings.ReplaceAll(parts[0], ".", "/")
	result := "L" + className + ";"

	if len(parts) == 2 {
		methodPart := parts[1]
		if paren := strings.Index(methodPart, "("); paren != -1 {
			result += "->" + methodPart[:paren] + javaParamsToDex(methodPart[paren:])
		} else {
			result += "->" + methodPart
		}
	}

	return result
}

// javaParamsToDex converts Java-style parameter list to DEX-style.
// "(java.lang.String, long, float, android.location.LocationListener)" → "(Ljava/lang/String;JFLandroid/location/LocationListener;)V"
func javaParamsToDex(params string) string {
	// Remove outer parens and whitespace
	params = strings.TrimSpace(params)
	if !strings.HasPrefix(params, "(") {
		return params
	}

	// Find closing paren
	closeParen := strings.LastIndex(params, ")")
	if closeParen == -1 {
		return params
	}

	inner := params[1:closeParen]
	// Return type (after closing paren)
	retPart := strings.TrimSpace(params[closeParen+1:])

	var dexParams strings.Builder
	dexParams.WriteByte('(')

	if strings.TrimSpace(inner) != "" {
		// Split by comma
		paramList := strings.Split(inner, ",")
		for _, p := range paramList {
			p = strings.TrimSpace(p)
			// Remove parameter names: "java.lang.String name" → "java.lang.String"
			// Take only the type (first token, or the full thing if no space, handling arrays)
			if spaceIdx := strings.LastIndex(p, " "); spaceIdx != -1 {
				// Check it's not part of the type name (shouldn't happen with Java types)
				candidate := strings.TrimSpace(p[:spaceIdx])
				// If candidate looks like a type, use it
				if candidate != "" && !strings.Contains(candidate, " ") {
					p = candidate
				}
			}
			dexParams.WriteString(javaTypeToDex(p))
		}
	}

	dexParams.WriteByte(')')

	if retPart != "" {
		dexParams.WriteString(javaTypeToDex(retPart))
	} else {
		dexParams.WriteByte('V') // default void
	}

	return dexParams.String()
}

// javaTypeToDex converts a single Java type to DEX descriptor.
func javaTypeToDex(javaType string) string {
	javaType = strings.TrimSpace(javaType)

	// Handle arrays
	arrayDims := 0
	for strings.HasSuffix(javaType, "[]") {
		arrayDims++
		javaType = javaType[:len(javaType)-2]
	}

	var desc string
	switch javaType {
	case "void":
		desc = "V"
	case "boolean":
		desc = "Z"
	case "byte":
		desc = "B"
	case "char":
		desc = "C"
	case "short":
		desc = "S"
	case "int":
		desc = "I"
	case "long":
		desc = "J"
	case "float":
		desc = "F"
	case "double":
		desc = "D"
	default:
		// Object type
		desc = "L" + strings.ReplaceAll(javaType, ".", "/") + ";"
	}

	// Prepend array dimensions
	prefix := strings.Repeat("[", arrayDims)
	return prefix + desc
}

// dexToJavaStyle converts a DEX-style signature back to Java-style (for display/matching).
func dexToJavaStyle(dexSig string) string {
	result := dexSig
	// "Lcom/foo/Bar;" → "com.foo.Bar"
	if strings.HasPrefix(result, "L") {
		if semi := strings.Index(result, ";"); semi != -1 {
			prefix := result[1:semi]
			prefix = strings.ReplaceAll(prefix, "/", ".")
			rest := result[semi+1:]
			result = prefix + rest
		}
	}
	result = strings.ReplaceAll(result, "/", ".")
	return result
}

// Query searches through scan results with flexible matching.
func Query(result *ScanResult, dexFiles []*dex.DexFile, query string, scope QueryScope) *QueryResult {
	qr := &QueryResult{
		MatchedMethods: make(map[string][]MethodRef),
		MatchedCallers: make(map[string][]MethodRef),
		MatchedFields:  make(map[string][]FieldRef),
		MatchedStrings: make(map[string][]StringRef),
	}

	if query == "" {
		qr.MatchedMethods = result.MethodRefs
		qr.MatchedFields = result.FieldRefs
		return qr
	}

	matcher := newQueryMatcher(query)

	// Search method references (callee)
	if scope&ScopeCallee != 0 {
		for api, refs := range result.MethodRefs {
			if matcher.matches(api) {
				qr.MatchedMethods[api] = refs
			}
		}
	}

	// Search callers
	if scope&ScopeCaller != 0 {
		for api, refs := range result.MethodRefs {
			for _, ref := range refs {
				if ref.CallerDexIdx < len(dexFiles) {
					callerName := dexFiles[ref.CallerDexIdx].GetApiMethodName(ref.CallerMethod)
					if matcher.matches(callerName) {
						qr.MatchedCallers[api] = append(qr.MatchedCallers[api], ref)
					}
				}
			}
		}
	}

	// Search field references
	if scope&ScopeCallee != 0 {
		for api, refs := range result.FieldRefs {
			if matcher.matches(api) {
				qr.MatchedFields[api] = refs
			}
		}
	}

	// Search string constants in code (const-string instructions)
	if scope&ScopeString != 0 {
		for str, refs := range result.StringRefs {
			if matcher.matches(str) {
				qr.MatchedStrings[str] = refs
			}
		}
	}

	// Search full DEX string table (catches strings in annotations, static init values, debug info)
	if scope&ScopeStringTable != 0 && result.AllStrings != nil {
		codeStrings := make(map[string]bool)
		for str := range qr.MatchedStrings {
			codeStrings[str] = true
		}
		for str := range result.AllStrings {
			if matcher.matches(str) && !codeStrings[str] {
				qr.MatchedStringTable = append(qr.MatchedStringTable, str)
			}
		}
	}

	return qr
}
