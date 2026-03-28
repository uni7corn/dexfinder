package report

import (
	"strings"

	"dex_method_finder/pkg/mapping"
)

// OutputFormat controls the output style.
type OutputFormat string

const (
	FormatText       OutputFormat = "text"       // DEX signatures + tree (--trace)
	FormatJSON       OutputFormat = "json"       // simple JSON
	FormatStacktrace OutputFormat = "stacktrace" // Java crash style, flat list of chains
	FormatTree       OutputFormat = "tree"       // Java readable names + tree (merged paths)
	FormatList       OutputFormat = "list"       // Java readable names + flat list of chains
	FormatModel      OutputFormat = "model"      // structured JSON for IDE/CI
)

// DisplayConfig controls how API signatures are displayed.
type DisplayConfig struct {
	Mapping *mapping.ProguardMapping
	ShowObf bool         // show obfuscated name alongside deobfuscated
	Format  OutputFormat // output format
}

// FormatAPI formats a DEX API signature for display.
func (dc *DisplayConfig) FormatAPI(dexSig string) string {
	if dc == nil || dc.Mapping == nil {
		return dexSig
	}

	deobf := dc.Mapping.DeobfuscateDexSignature(dexSig)
	if deobf == dexSig {
		return dexSig
	}

	if dc.ShowObf {
		return deobf + "  ← " + shortObfName(dexSig)
	}
	return deobf
}

// FormatShort formats for tree display (shorter names).
func (dc *DisplayConfig) FormatShort(dexSig string) string {
	if dc == nil || dc.Mapping == nil {
		return shortName(dexSig)
	}

	deobf := dc.Mapping.DeobfuscateDexSignature(dexSig)
	short := shortName(deobf)

	if dc.ShowObf && deobf != dexSig {
		return short + "  ← " + shortName(dexSig)
	}
	return short
}

// FormatStacktraceLine formats a DEX signature as a Java stacktrace line.
// "Lcom/foo/Bar;->method(Ljava/lang/String;)V"
// → "com.foo.Bar.method(Bar.java)"
// With mapping:
// → "com.original.Class.origMethod(Class.java)"
// With --show-obf:
// → "com.original.Class.origMethod(Class.java) [obf: a.b.c]"
func (dc *DisplayConfig) FormatStacktraceLine(dexSig string) string {
	sig := dexSig
	if dc != nil && dc.Mapping != nil {
		sig = dc.Mapping.DeobfuscateDexSignature(dexSig)
	}

	javaStyle := dexToJavaStacktrace(sig)

	if dc != nil && dc.ShowObf && sig != dexSig {
		return javaStyle + "  [obf: " + dexToJavaCompact(dexSig) + "]"
	}
	return javaStyle
}

// FormatStacktraceTarget formats the target API for stacktrace header.
// "Lcom/foo/Bar;->method(Ljava/lang/String;)V"
// → "com.foo.Bar.method(String)"
func (dc *DisplayConfig) FormatStacktraceTarget(dexSig string) string {
	sig := dexSig
	if dc != nil && dc.Mapping != nil {
		sig = dc.Mapping.DeobfuscateDexSignature(dexSig)
	}
	return dexToJavaReadable(sig)
}

// --- Converters ---

// dexToJavaStacktrace: "Lcom/foo/Bar;->method(Ljava/lang/String;)V" → "com.foo.Bar.method(Bar.java)"
func dexToJavaStacktrace(dexSig string) string {
	arrowIdx := strings.Index(dexSig, "->")
	if arrowIdx == -1 {
		return dexClassToJavaDot(dexSig)
	}

	classDesc := dexSig[:arrowIdx]
	member := dexSig[arrowIdx+2:]

	className := dexClassToJavaDot(classDesc)
	simpleClass := className
	if dotIdx := strings.LastIndex(simpleClass, "."); dotIdx != -1 {
		simpleClass = simpleClass[dotIdx+1:]
	}
	// Handle inner class: "Foo$Bar" → "Foo" for file name
	sourceFile := simpleClass
	if dollarIdx := strings.Index(sourceFile, "$"); dollarIdx != -1 {
		sourceFile = sourceFile[:dollarIdx]
	}

	// Extract method name (strip params and return type)
	methodName := member
	if parenIdx := strings.Index(member, "("); parenIdx != -1 {
		methodName = member[:parenIdx]
	}

	return className + "." + methodName + "(" + sourceFile + ".java)"
}

// dexToJavaReadable: "Lcom/foo/Bar;->method(Ljava/lang/String;JF)V"
// → "com.foo.Bar.method(String, long, float)"
func dexToJavaReadable(dexSig string) string {
	arrowIdx := strings.Index(dexSig, "->")
	if arrowIdx == -1 {
		return dexClassToJavaDot(dexSig)
	}

	classDesc := dexSig[:arrowIdx]
	member := dexSig[arrowIdx+2:]
	className := dexClassToJavaDot(classDesc)

	parenOpen := strings.Index(member, "(")
	if parenOpen == -1 {
		// Field
		return className + "." + member
	}

	methodName := member[:parenOpen]
	parenClose := strings.LastIndex(member, ")")
	if parenClose == -1 {
		return className + "." + member
	}

	dexParams := member[parenOpen+1 : parenClose]
	javaParams := dexParamsToJavaReadable(dexParams)

	return className + "." + methodName + "(" + javaParams + ")"
}

// dexToJavaCompact: for [obf: ...] display — short form
func dexToJavaCompact(dexSig string) string {
	arrowIdx := strings.Index(dexSig, "->")
	if arrowIdx == -1 {
		return dexClassToJavaDot(dexSig)
	}

	classDesc := dexSig[:arrowIdx]
	member := dexSig[arrowIdx+2:]
	className := dexClassToJavaDot(classDesc)

	// Just class.method (no params)
	methodName := member
	if parenIdx := strings.Index(member, "("); parenIdx != -1 {
		methodName = member[:parenIdx]
	}
	return className + "." + methodName
}

// dexClassToJavaDot: "Lcom/foo/Bar;" → "com.foo.Bar"
func dexClassToJavaDot(dexClass string) string {
	s := dexClass
	s = strings.TrimPrefix(s, "L")
	s = strings.TrimSuffix(s, ";")
	return strings.ReplaceAll(s, "/", ".")
}

// dexParamsToJavaReadable: "Ljava/lang/String;JFLandroid/location/LocationListener;"
// → "String, long, float, LocationListener"
func dexParamsToJavaReadable(dexParams string) string {
	if dexParams == "" {
		return ""
	}

	var parts []string
	i := 0
	for i < len(dexParams) {
		arrayDims := 0
		for i < len(dexParams) && dexParams[i] == '[' {
			arrayDims++
			i++
		}
		if i >= len(dexParams) {
			break
		}

		var typeName string
		switch dexParams[i] {
		case 'V':
			typeName = "void"
			i++
		case 'Z':
			typeName = "boolean"
			i++
		case 'B':
			typeName = "byte"
			i++
		case 'C':
			typeName = "char"
			i++
		case 'S':
			typeName = "short"
			i++
		case 'I':
			typeName = "int"
			i++
		case 'J':
			typeName = "long"
			i++
		case 'F':
			typeName = "float"
			i++
		case 'D':
			typeName = "double"
			i++
		case 'L':
			// Object type: find semicolon
			semi := strings.Index(dexParams[i:], ";")
			if semi == -1 {
				typeName = dexParams[i:]
				i = len(dexParams)
			} else {
				fullClass := dexParams[i+1 : i+semi]
				// Use simple class name only
				if lastSlash := strings.LastIndex(fullClass, "/"); lastSlash != -1 {
					typeName = strings.ReplaceAll(fullClass[lastSlash+1:], "$", ".")
				} else {
					typeName = fullClass
				}
				i = i + semi + 1
			}
		default:
			i++
			continue
		}

		for j := 0; j < arrayDims; j++ {
			typeName += "[]"
		}
		parts = append(parts, typeName)
	}

	return strings.Join(parts, ", ")
}

// shortObfName extracts a shorter display name from the obfuscated signature.
func shortObfName(fullAPI string) string {
	arrowIdx := strings.Index(fullAPI, "->")
	if arrowIdx == -1 {
		return fullAPI
	}
	classDesc := fullAPI[:arrowIdx]
	member := fullAPI[arrowIdx+2:]

	className := classDesc
	if strings.HasPrefix(className, "L") && strings.HasSuffix(className, ";") {
		className = className[1 : len(className)-1]
	}
	if lastSlash := strings.LastIndex(className, "/"); lastSlash != -1 {
		className = className[lastSlash+1:]
	}
	return className + "." + member
}
