package mapping

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ProguardMapping holds bidirectional class/method/field name mappings.
type ProguardMapping struct {
	// original class → obfuscated class
	classToObf map[string]string
	// obfuscated class → original class
	obfToClass map[string]string

	// "OrigClass.origMethod(args)retType" → "ObfClass.obfMethod"
	methodToObf map[string]string
	// "ObfClass.obfMethod" → []original (multiple due to overloads)
	obfToMethods map[string][]MethodMapping

	// "OrigClass.origField" → "ObfClass.obfField"
	fieldToObf map[string]string
	// "ObfClass.obfField" → "OrigClass.origField"
	obfToField map[string]string
}

// MethodMapping stores one method mapping entry.
type MethodMapping struct {
	OriginalClass  string
	OriginalName   string
	OriginalArgs   string // Java-style: "int,java.lang.String"
	OriginalReturn string
	ObfClass       string
	ObfName        string
}

// FullOriginal returns "OrigClass.origMethod(args)retType"
func (m *MethodMapping) FullOriginal() string {
	return m.OriginalClass + "." + m.OriginalName + "(" + m.OriginalArgs + ")" + m.OriginalReturn
}

// OrigDexSignature returns DEX-style: "Lcom/foo/Bar;->method(Ljava/lang/String;)V"
func (m *MethodMapping) OrigDexSignature() string {
	cls := javaClassToDex(m.OriginalClass)
	args := javaArgsToDexArgs(m.OriginalArgs)
	ret := javaTypeToDexType(m.OriginalReturn)
	return cls + "->" + m.OriginalName + "(" + args + ")" + ret
}

// ObfDexClass returns DEX-style class descriptor for the obfuscated class.
func (m *MethodMapping) ObfDexClass() string {
	return javaClassToDex(m.ObfClass)
}

// LoadProguardMapping parses a ProGuard/R8 mapping.txt file.
func LoadProguardMapping(filename string) (*ProguardMapping, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open mapping file: %w", err)
	}
	defer f.Close()

	pm := &ProguardMapping{
		classToObf:   make(map[string]string),
		obfToClass:   make(map[string]string),
		methodToObf:  make(map[string]string),
		obfToMethods: make(map[string][]MethodMapping),
		fieldToObf:   make(map[string]string),
		obfToField:   make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var currentOrigClass string
	var currentObfClass string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Class mapping: "com.example.Foo -> a.b:"
			origClass, obfClass, ok := parseClassLine(line)
			if !ok {
				continue
			}
			currentOrigClass = origClass
			currentObfClass = obfClass
			pm.classToObf[origClass] = obfClass
			pm.obfToClass[obfClass] = origClass
		} else {
			// Member mapping (indented)
			line = strings.TrimSpace(line)
			if currentOrigClass == "" {
				continue
			}

			if strings.Contains(line, "(") {
				// Method mapping
				mm := parseMethodLine(line, currentOrigClass, currentObfClass)
				if mm != nil {
					key := mm.FullOriginal()
					obfKey := currentObfClass + "." + mm.ObfName
					pm.methodToObf[key] = obfKey
					pm.obfToMethods[obfKey] = append(pm.obfToMethods[obfKey], *mm)
				}
			} else {
				// Field mapping
				origField, obfField := parseFieldLine(line)
				if origField != "" {
					origKey := currentOrigClass + "." + origField
					obfKey := currentObfClass + "." + obfField
					pm.fieldToObf[origKey] = obfKey
					pm.obfToField[obfKey] = origKey
				}
			}
		}
	}

	return pm, scanner.Err()
}

// parseClassLine: "com.example.Foo -> a.b:" → ("com.example.Foo", "a.b")
func parseClassLine(line string) (string, string, bool) {
	idx := strings.Index(line, " -> ")
	if idx == -1 {
		return "", "", false
	}
	orig := line[:idx]
	rest := line[idx+4:]
	obf := strings.TrimSuffix(rest, ":")
	return strings.TrimSpace(orig), strings.TrimSpace(obf), true
}

// parseMethodLine: "    1:5:void origMethod(int,String):15:19 -> d"
// or: "    void origMethod(int,String) -> d"
func parseMethodLine(line string, origClass, obfClass string) *MethodMapping {
	arrowIdx := strings.LastIndex(line, " -> ")
	if arrowIdx == -1 {
		return nil
	}
	obfName := strings.TrimSpace(line[arrowIdx+4:])
	left := strings.TrimSpace(line[:arrowIdx])

	// Strip line number info: "1:5:void method(args):15:19" → "void method(args)"
	// Format: optional "startLine:endLine:" prefix, optional ":sourceStart:sourceEnd" suffix
	left = stripLineNumbers(left)

	// Parse "retType methodName(argTypes)"
	parenOpen := strings.Index(left, "(")
	parenClose := strings.LastIndex(left, ")")
	if parenOpen == -1 || parenClose == -1 {
		return nil
	}

	beforeParen := left[:parenOpen]
	args := left[parenOpen+1 : parenClose]

	// beforeParen = "void methodName" or "int methodName"
	spaceIdx := strings.LastIndex(beforeParen, " ")
	if spaceIdx == -1 {
		// Constructor: "<init>(args)" — no return type
		return &MethodMapping{
			OriginalClass:  origClass,
			OriginalName:   beforeParen,
			OriginalArgs:   args,
			OriginalReturn: "void",
			ObfClass:       obfClass,
			ObfName:        obfName,
		}
	}

	retType := strings.TrimSpace(beforeParen[:spaceIdx])
	methodName := strings.TrimSpace(beforeParen[spaceIdx+1:])

	return &MethodMapping{
		OriginalClass:  origClass,
		OriginalName:   methodName,
		OriginalArgs:   args,
		OriginalReturn: retType,
		ObfClass:       obfClass,
		ObfName:        obfName,
	}
}

// stripLineNumbers removes "1:5:" prefix and ":15:19" suffix from method descriptor.
func stripLineNumbers(s string) string {
	// Strip prefix like "1:5:"
	for {
		colonIdx := strings.Index(s, ":")
		if colonIdx == -1 {
			break
		}
		// Check if everything before colon is digits
		prefix := s[:colonIdx]
		if !isDigits(prefix) {
			break
		}
		s = s[colonIdx+1:]
	}

	// Strip suffix like ":15:19" — find the last paren then check for trailing ":digits:digits"
	parenClose := strings.LastIndex(s, ")")
	if parenClose != -1 && parenClose < len(s)-1 {
		suffix := s[parenClose+1:]
		if strings.HasPrefix(suffix, ":") {
			s = s[:parenClose+1]
		}
	}

	return s
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseFieldLine: "int origField -> c" → ("origField", "c")
func parseFieldLine(line string) (string, string) {
	arrowIdx := strings.LastIndex(line, " -> ")
	if arrowIdx == -1 {
		return "", ""
	}
	obfName := strings.TrimSpace(line[arrowIdx+4:])
	left := strings.TrimSpace(line[:arrowIdx])

	// "int origField" → "origField"
	spaceIdx := strings.LastIndex(left, " ")
	if spaceIdx == -1 {
		return left, obfName
	}
	return strings.TrimSpace(left[spaceIdx+1:]), obfName
}

// --- Public API ---

// DeobfuscateClass returns the original class name for an obfuscated one.
func (pm *ProguardMapping) DeobfuscateClass(obfClass string) string {
	// Try direct lookup
	if orig, ok := pm.obfToClass[obfClass]; ok {
		return orig
	}
	return obfClass
}

// ObfuscateClass returns the obfuscated class name for an original one.
func (pm *ProguardMapping) ObfuscateClass(origClass string) string {
	if obf, ok := pm.classToObf[origClass]; ok {
		return obf
	}
	return origClass
}

// DeobfuscateDexSignature converts an obfuscated DEX signature to original.
// "La/b;->c(I)V" → "Lcom/example/Foo;->originalMethod(I)V"
func (pm *ProguardMapping) DeobfuscateDexSignature(dexSig string) string {
	arrowIdx := strings.Index(dexSig, "->")
	if arrowIdx == -1 {
		// Just a class descriptor
		return pm.deobfDexClass(dexSig)
	}

	obfClassDex := dexSig[:arrowIdx]
	member := dexSig[arrowIdx+2:]

	origClassDex := pm.deobfDexClass(obfClassDex)
	obfClassJava := dexClassToJava(obfClassDex)

	// Try to find method name
	parenIdx := strings.Index(member, "(")
	if parenIdx != -1 {
		obfMethodName := member[:parenIdx]
		obfKey := obfClassJava + "." + obfMethodName
		if methods, ok := pm.obfToMethods[obfKey]; ok && len(methods) > 0 {
			// Use first match (could be ambiguous with overloads)
			origMethod := methods[0].OriginalName
			return origClassDex + "->" + origMethod + member[parenIdx:]
		}
	} else {
		// Field: member is "fieldName:Type"
		colonIdx := strings.Index(member, ":")
		var obfFieldName string
		var suffix string
		if colonIdx != -1 {
			obfFieldName = member[:colonIdx]
			suffix = member[colonIdx:]
		} else {
			obfFieldName = member
		}
		obfKey := obfClassJava + "." + obfFieldName
		if origKey, ok := pm.obfToField[obfKey]; ok {
			// "com.example.Foo.origField" → "origField"
			dotIdx := strings.LastIndex(origKey, ".")
			origField := origKey[dotIdx+1:]
			return origClassDex + "->" + origField + suffix
		}
	}

	return origClassDex + "->" + member
}

// ObfuscateDexSignature converts an original DEX signature to obfuscated.
// "Lcom/example/Foo;->originalMethod(I)V" → "La/b;->c(I)V"
func (pm *ProguardMapping) ObfuscateDexSignature(dexSig string) string {
	arrowIdx := strings.Index(dexSig, "->")
	if arrowIdx == -1 {
		return pm.obfDexClass(dexSig)
	}

	origClassDex := dexSig[:arrowIdx]
	member := dexSig[arrowIdx+2:]

	obfClassDex := pm.obfDexClass(origClassDex)
	origClassJava := dexClassToJava(origClassDex)

	parenIdx := strings.Index(member, "(")
	if parenIdx != -1 {
		origMethodName := member[:parenIdx]
		// Search all methods of this class for matching name
		for key, obfKey := range pm.methodToObf {
			if strings.HasPrefix(key, origClassJava+"."+origMethodName+"(") {
				// Found: extract obfuscated method name
				dotIdx := strings.LastIndex(obfKey, ".")
				obfMethod := obfKey[dotIdx+1:]
				return obfClassDex + "->" + obfMethod + member[parenIdx:]
			}
		}
	} else {
		colonIdx := strings.Index(member, ":")
		var origFieldName string
		var suffix string
		if colonIdx != -1 {
			origFieldName = member[:colonIdx]
			suffix = member[colonIdx:]
		} else {
			origFieldName = member
		}
		origKey := origClassJava + "." + origFieldName
		if obfKey, ok := pm.fieldToObf[origKey]; ok {
			dotIdx := strings.LastIndex(obfKey, ".")
			obfField := obfKey[dotIdx+1:]
			return obfClassDex + "->" + obfField + suffix
		}
	}

	return obfClassDex + "->" + member
}

// OriginalClassForDex returns the original Java class name for a DEX class descriptor.
func (pm *ProguardMapping) OriginalClassForDex(dexClass string) string {
	javaClass := dexClassToJava(dexClass)
	if orig, ok := pm.obfToClass[javaClass]; ok {
		return orig
	}
	return javaClass
}

// Size returns the number of class mappings.
func (pm *ProguardMapping) Size() int {
	return len(pm.classToObf)
}

// --- Internal helpers ---

func (pm *ProguardMapping) deobfDexClass(dexClass string) string {
	javaClass := dexClassToJava(dexClass)
	if orig, ok := pm.obfToClass[javaClass]; ok {
		return javaClassToDex(orig)
	}
	return dexClass
}

func (pm *ProguardMapping) obfDexClass(dexClass string) string {
	javaClass := dexClassToJava(dexClass)
	if obf, ok := pm.classToObf[javaClass]; ok {
		return javaClassToDex(obf)
	}
	return dexClass
}

// javaClassToDex: "com.example.Foo" → "Lcom/example/Foo;"
func javaClassToDex(javaClass string) string {
	return "L" + strings.ReplaceAll(javaClass, ".", "/") + ";"
}

// dexClassToJava: "Lcom/example/Foo;" → "com.example.Foo"
func dexClassToJava(dexClass string) string {
	s := dexClass
	s = strings.TrimPrefix(s, "L")
	s = strings.TrimSuffix(s, ";")
	return strings.ReplaceAll(s, "/", ".")
}

// javaArgsToDexArgs: "int,java.lang.String" → "ILjava/lang/String;"
func javaArgsToDexArgs(args string) string {
	if args == "" {
		return ""
	}
	var sb strings.Builder
	for _, arg := range strings.Split(args, ",") {
		sb.WriteString(javaTypeToDexType(strings.TrimSpace(arg)))
	}
	return sb.String()
}

// javaTypeToDexType: "int" → "I", "java.lang.String" → "Ljava/lang/String;"
func javaTypeToDexType(javaType string) string {
	javaType = strings.TrimSpace(javaType)

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
		desc = "L" + strings.ReplaceAll(javaType, ".", "/") + ";"
	}

	return strings.Repeat("[", arrayDims) + desc
}
