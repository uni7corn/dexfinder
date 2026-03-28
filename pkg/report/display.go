package report

import (
	"strings"

	"dex_method_finder/pkg/mapping"
)

// OutputFormat controls the output style.
type OutputFormat string

const (
	FormatText  OutputFormat = "text"  // DEX signatures (default)
	FormatJSON  OutputFormat = "json"  // simple JSON
	FormatModel OutputFormat = "model" // structured JSON for IDE/CI
)

// TraceLayout controls how call chains are rendered.
type TraceLayout string

const (
	LayoutTree TraceLayout = "tree" // merged tree (shared paths collapsed)
	LayoutList TraceLayout = "list" // flat list of individual chains
)

// NameStyle controls how method/class names are displayed.
type NameStyle string

const (
	StyleDex  NameStyle = "dex"  // DEX/JNI signature: Lcom/foo/Bar;->method(I)V
	StyleJava NameStyle = "java" // Java readable: com.foo.Bar.method(Bar.java)
)

// DisplayConfig controls how API signatures are displayed.
type DisplayConfig struct {
	Mapping *mapping.ProguardMapping
	ShowObf bool         // show obfuscated name alongside deobfuscated
	Format  OutputFormat // output format (text/json/model)
	Layout  TraceLayout  // tree or list (for --trace)
	Style   NameStyle    // dex or java (name display style)
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

// FormatNode formats a call chain node based on the NameStyle.
func (dc *DisplayConfig) FormatNode(dexSig string) string {
	if dc != nil && dc.Style == StyleJava {
		return dc.FormatStacktraceLine(dexSig)
	}
	return dc.FormatShort(dexSig)
}

// FormatHeader formats the target API header based on NameStyle.
func (dc *DisplayConfig) FormatHeader(dexSig string) string {
	if dc != nil && dc.Style == StyleJava {
		return dc.FormatStacktraceTarget(dexSig)
	}
	return dc.FormatAPI(dexSig)
}

// FormatStacktraceLine formats a DEX signature as a Java stacktrace line.
// "Lcom/foo/Bar;->method(Ljava/lang/String;)V" → "com.foo.Bar.method(Bar.java)"
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
// "Lcom/foo/Bar;->method(Ljava/lang/String;)V" → "com.foo.Bar.method(String)"
func (dc *DisplayConfig) FormatStacktraceTarget(dexSig string) string {
	sig := dexSig
	if dc != nil && dc.Mapping != nil {
		sig = dc.Mapping.DeobfuscateDexSignature(dexSig)
	}
	return dexToJavaReadable(sig)
}

// --- Converters ---

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
	sourceFile := simpleClass
	if dollarIdx := strings.Index(sourceFile, "$"); dollarIdx != -1 {
		sourceFile = sourceFile[:dollarIdx]
	}

	methodName := member
	if parenIdx := strings.Index(member, "("); parenIdx != -1 {
		methodName = member[:parenIdx]
	}

	return className + "." + methodName + "(" + sourceFile + ".java)"
}

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

func dexToJavaCompact(dexSig string) string {
	arrowIdx := strings.Index(dexSig, "->")
	if arrowIdx == -1 {
		return dexClassToJavaDot(dexSig)
	}
	classDesc := dexSig[:arrowIdx]
	member := dexSig[arrowIdx+2:]
	className := dexClassToJavaDot(classDesc)
	methodName := member
	if parenIdx := strings.Index(member, "("); parenIdx != -1 {
		methodName = member[:parenIdx]
	}
	return className + "." + methodName
}

func dexClassToJavaDot(dexClass string) string {
	s := dexClass
	s = strings.TrimPrefix(s, "L")
	s = strings.TrimSuffix(s, ";")
	return strings.ReplaceAll(s, "/", ".")
}

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
			typeName = "void"; i++
		case 'Z':
			typeName = "boolean"; i++
		case 'B':
			typeName = "byte"; i++
		case 'C':
			typeName = "char"; i++
		case 'S':
			typeName = "short"; i++
		case 'I':
			typeName = "int"; i++
		case 'J':
			typeName = "long"; i++
		case 'F':
			typeName = "float"; i++
		case 'D':
			typeName = "double"; i++
		case 'L':
			semi := strings.Index(dexParams[i:], ";")
			if semi == -1 {
				typeName = dexParams[i:]
				i = len(dexParams)
			} else {
				fullClass := dexParams[i+1 : i+semi]
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
