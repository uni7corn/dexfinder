package model

import (
	"strings"

	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/mapping"
)

// Converter transforms raw scan results into the structured model.
type Converter struct {
	DexFiles []*dex.DexFile
	Mapping  *mapping.ProguardMapping
}

// BuildMethodInfo creates a MethodInfo from a DEX file and method index.
func (c *Converter) BuildMethodInfo(dexIdx int, methodIdx uint32) MethodInfo {
	df := c.DexFiles[dexIdx]
	dexSig := df.GetApiMethodName(methodIdx)
	mi := MethodInfo{
		DexSignature: dexSig,
		DexIndex:     dexIdx,
		MethodIndex:  methodIdx,
	}

	// Parse class, name, params, return type from DEX signature
	if arrowIdx := strings.Index(dexSig, "->"); arrowIdx != -1 {
		mi.Class = dexSig[:arrowIdx]
		member := dexSig[arrowIdx+2:]
		if parenOpen := strings.Index(member, "("); parenOpen != -1 {
			mi.Name = member[:parenOpen]
			if parenClose := strings.LastIndex(member, ")"); parenClose != -1 {
				mi.ReturnType = member[parenClose+1:]
				mi.ParamTypes = parseDexParamTypes(member[parenOpen+1 : parenClose])
			}
		}
	}

	mi.JavaReadable = dexToJavaReadable(dexSig)

	if c.Mapping != nil {
		deobf := c.Mapping.DeobfuscateDexSignature(dexSig)
		if deobf != dexSig {
			mi.OriginalSignature = deobf
		}
	}

	return mi
}

// BuildFieldInfo creates a FieldInfo from a DEX field signature.
func (c *Converter) BuildFieldInfo(dexIdx int, fieldIdx uint32) FieldInfo {
	df := c.DexFiles[dexIdx]
	dexSig := df.GetApiFieldName(fieldIdx)
	fi := FieldInfo{
		DexSignature: dexSig,
		DexIndex:     dexIdx,
		FieldIndex:   fieldIdx,
	}

	if arrowIdx := strings.Index(dexSig, "->"); arrowIdx != -1 {
		fi.Class = dexSig[:arrowIdx]
		member := dexSig[arrowIdx+2:]
		if colonIdx := strings.Index(member, ":"); colonIdx != -1 {
			fi.Name = member[:colonIdx]
			fi.Type = member[colonIdx+1:]
		} else {
			fi.Name = member
		}
	}

	if c.Mapping != nil {
		deobf := c.Mapping.DeobfuscateDexSignature(dexSig)
		if deobf != dexSig {
			fi.OriginalSignature = deobf
		}
	}

	return fi
}

// BuildLocation creates a Location from a method reference.
func (c *Converter) BuildLocation(dexIdx int, methodIdx uint32) Location {
	return Location{
		Method: c.BuildMethodInfo(dexIdx, methodIdx),
	}
}

// ConvertScanResult transforms a ScanResult into the structured AnalysisResult.
func (c *Converter) ConvertScanResult(result *finder.ScanResult, meta Metadata) *AnalysisResult {
	ar := &AnalysisResult{
		Metadata: meta,
	}

	// Method calls
	for api, refs := range result.MethodRefs {
		mci := MethodCallInfo{
			Target: c.parseTargetMethod(api),
			Count:  len(refs),
		}
		seen := make(map[string]bool)
		for _, ref := range refs {
			key := dexMethodKey(ref.CallerDexIdx, ref.CallerMethod)
			if seen[key] {
				continue
			}
			seen[key] = true
			mci.Locations = append(mci.Locations, c.BuildLocation(ref.CallerDexIdx, ref.CallerMethod))
		}
		ar.MethodCalls = append(ar.MethodCalls, mci)
	}

	// Field accesses
	for api, refs := range result.FieldRefs {
		fai := FieldAccessInfo{
			Target: c.parseTargetField(api),
			Count:  len(refs),
		}
		seen := make(map[string]bool)
		for _, ref := range refs {
			key := dexMethodKey(ref.CallerDexIdx, ref.CallerMethod)
			if seen[key] {
				continue
			}
			seen[key] = true
			fai.Locations = append(fai.Locations, c.BuildLocation(ref.CallerDexIdx, ref.CallerMethod))
		}
		ar.FieldAccesses = append(ar.FieldAccesses, fai)
	}

	// String refs
	for str, refs := range result.StringRefs {
		sri := StringRefInfo{
			Value: str,
			Count: len(refs),
		}
		seen := make(map[string]bool)
		for _, ref := range refs {
			key := dexMethodKey(ref.CallerDexIdx, ref.CallerMethod)
			if seen[key] {
				continue
			}
			seen[key] = true
			sri.Locations = append(sri.Locations, c.BuildLocation(ref.CallerDexIdx, ref.CallerMethod))
		}
		ar.StringRefs = append(ar.StringRefs, sri)
	}

	// Summary
	ar.Summary = Summary{
		TotalClasses:     len(result.Classes),
		TotalMethodCalls: len(result.MethodRefs),
		TotalFieldAccess: len(result.FieldRefs),
		TotalStrings:     len(result.StringRefs),
	}

	return ar
}

// ConvertCallChains converts traced call chains to the structured model.
func (c *Converter) ConvertCallChains(root *finder.CallChainNode) []CallChainInfo {
	flat := finder.FlatCallerChains(root, 100)
	var chains []CallChainInfo

	for _, chain := range flat {
		cci := CallChainInfo{
			Target: chain[0],
			Depth:  len(chain) - 1,
		}
		// Reverse: chain[0]=target, chain[last]=root → output root first
		for j := len(chain) - 1; j >= 0; j-- {
			entry := CallChainEntry{
				Method: c.parseTargetMethod(chain[j]),
			}
			if strings.HasSuffix(chain[j], " [recursive]") {
				entry.IsCycle = true
				entry.Method = c.parseTargetMethod(strings.TrimSuffix(chain[j], " [recursive]"))
			}
			cci.Chain = append(cci.Chain, entry)
		}
		chains = append(chains, cci)
	}

	return chains
}

func (c *Converter) parseTargetMethod(dexSig string) MethodInfo {
	mi := MethodInfo{
		DexSignature: dexSig,
		JavaReadable: dexToJavaReadable(dexSig),
	}

	if arrowIdx := strings.Index(dexSig, "->"); arrowIdx != -1 {
		mi.Class = dexSig[:arrowIdx]
		member := dexSig[arrowIdx+2:]
		if parenOpen := strings.Index(member, "("); parenOpen != -1 {
			mi.Name = member[:parenOpen]
			if parenClose := strings.LastIndex(member, ")"); parenClose != -1 {
				mi.ReturnType = member[parenClose+1:]
				mi.ParamTypes = parseDexParamTypes(member[parenOpen+1 : parenClose])
			}
		}
	}

	if c.Mapping != nil {
		deobf := c.Mapping.DeobfuscateDexSignature(dexSig)
		if deobf != dexSig {
			mi.OriginalSignature = deobf
		}
	}

	return mi
}

func (c *Converter) parseTargetField(dexSig string) FieldInfo {
	fi := FieldInfo{DexSignature: dexSig}
	if arrowIdx := strings.Index(dexSig, "->"); arrowIdx != -1 {
		fi.Class = dexSig[:arrowIdx]
		member := dexSig[arrowIdx+2:]
		if colonIdx := strings.Index(member, ":"); colonIdx != -1 {
			fi.Name = member[:colonIdx]
			fi.Type = member[colonIdx+1:]
		}
	}
	if c.Mapping != nil {
		deobf := c.Mapping.DeobfuscateDexSignature(dexSig)
		if deobf != dexSig {
			fi.OriginalSignature = deobf
		}
	}
	return fi
}

func dexMethodKey(dexIdx int, methodIdx uint32) string {
	return string(rune(dexIdx)) + ":" + string(rune(methodIdx))
}

// parseDexParamTypes splits DEX parameter descriptor into individual types.
// "Ljava/lang/String;JF" → ["Ljava/lang/String;", "J", "F"]
func parseDexParamTypes(params string) []string {
	if params == "" {
		return nil
	}
	var types []string
	i := 0
	for i < len(params) {
		start := i
		// Handle array prefix
		for i < len(params) && params[i] == '[' {
			i++
		}
		if i >= len(params) {
			break
		}
		switch params[i] {
		case 'L':
			semi := strings.Index(params[i:], ";")
			if semi == -1 {
				types = append(types, params[start:])
				i = len(params)
			} else {
				i = i + semi + 1
				types = append(types, params[start:i])
			}
		case 'V', 'Z', 'B', 'C', 'S', 'I', 'J', 'F', 'D':
			i++
			types = append(types, params[start:i])
		default:
			i++
		}
	}
	return types
}

// dexToJavaReadable converts DEX sig to human-readable form.
func dexToJavaReadable(dexSig string) string {
	arrowIdx := strings.Index(dexSig, "->")
	if arrowIdx == -1 {
		return dexClassToJava(dexSig)
	}
	cls := dexClassToJava(dexSig[:arrowIdx])
	member := dexSig[arrowIdx+2:]
	if parenOpen := strings.Index(member, "("); parenOpen != -1 {
		name := member[:parenOpen]
		return cls + "." + name + "(...)"
	}
	return cls + "." + member
}

func dexClassToJava(desc string) string {
	s := strings.TrimPrefix(desc, "L")
	s = strings.TrimSuffix(s, ";")
	return strings.ReplaceAll(s, "/", ".")
}
