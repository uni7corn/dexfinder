package finder

import (
	"strings"

	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/hiddenapi"
)

// MethodRef represents a reference from a caller to a callee method.
type MethodRef struct {
	CallerDexIdx int
	CallerMethod uint32 // method index of the caller
	CalleeAPI    string // full API signature of the callee
}

// FieldRef represents a reference from a caller to a field.
type FieldRef struct {
	CallerDexIdx int
	CallerMethod uint32
	FieldAPI     string
}

// StringRef represents a string constant found in code.
type StringRef struct {
	CallerDexIdx int
	CallerMethod uint32
	Value        string
}

// ScanResult holds all findings from scanning DEX files.
type ScanResult struct {
	MethodRefs map[string][]MethodRef // callee API → list of callers
	FieldRefs  map[string][]FieldRef  // field API → list of callers
	StringRefs map[string][]StringRef // string value → list of callers (from const-string in code)
	Classes    map[string]bool        // all referenced type descriptors
	AllStrings map[string]bool        // full DEX string table (includes annotations, debug info, etc.)
}

// DirectFinder scans DEX instructions to find method/field references.
type DirectFinder struct {
	dexFiles []*dex.DexFile
	filter   *ClassFilter
	db       *hiddenapi.Database // optional, nil for scan-only mode
}

// NewDirectFinder creates a finder for the given DEX files.
func NewDirectFinder(dexFiles []*dex.DexFile, filter *ClassFilter, db *hiddenapi.Database) *DirectFinder {
	return &DirectFinder{
		dexFiles: dexFiles,
		filter:   filter,
		db:       db,
	}
}

// Scan iterates all instructions and collects references.
func (f *DirectFinder) Scan() *ScanResult {
	result := &ScanResult{
		MethodRefs: make(map[string][]MethodRef),
		FieldRefs:  make(map[string][]FieldRef),
		StringRefs: make(map[string][]StringRef),
		Classes:    make(map[string]bool),
		AllStrings: make(map[string]bool),
	}

	for dexIdx, df := range f.dexFiles {
		// Collect all referenced types
		for i := uint32(0); i < df.NumTypeIDs(); i++ {
			desc := df.GetTypeDescriptor(i)
			result.Classes[desc] = true
		}

		// Collect full string table
		for i := uint32(0); i < df.NumStringIDs(); i++ {
			s := df.GetString(i)
			if s != "" {
				result.AllStrings[s] = true
			}
		}

		// Scan each class
		for ci := range df.ClassDefs {
			cd := &df.ClassDefs[ci]
			classDesc := df.GetTypeDescriptor(cd.ClassIdx)
			if !f.filter.Matches(classDesc) {
				continue
			}

			classData := df.GetClassData(cd)
			if classData == nil {
				continue
			}

			for _, method := range classData.AllMethods() {
				if method.CodeOff == 0 {
					continue
				}

				codeItem := df.GetCodeItem(method.CodeOff)
				if codeItem == nil || len(codeItem.Insns) == 0 {
					continue
				}

				f.scanMethod(dexIdx, df, method.MethodIdx, codeItem, result)
			}
		}
	}

	return result
}

func (f *DirectFinder) scanMethod(dexIdx int, df *dex.DexFile, methodIdx uint32, code *dex.CodeItem, result *ScanResult) {
	instructions := dex.DecodeAll(code.Insns)

	for i := range instructions {
		inst := &instructions[i]
		op := inst.Op

		switch {
		// Invoke instructions (non-range): method index in VRegB_35c
		case op == dex.OpInvokeVirtual || op == dex.OpInvokeSuper ||
			op == dex.OpInvokeDirect || op == dex.OpInvokeStatic ||
			op == dex.OpInvokeInterface:
			calleeIdx := inst.VRegB_35c()
			api := df.GetApiMethodName(calleeIdx)
			result.MethodRefs[api] = append(result.MethodRefs[api], MethodRef{
				CallerDexIdx: dexIdx,
				CallerMethod: methodIdx,
				CalleeAPI:    api,
			})

		// Invoke range instructions: method index in VRegB_3rc
		case op == dex.OpInvokeVirtualRange || op == dex.OpInvokeSuperRange ||
			op == dex.OpInvokeDirectRange || op == dex.OpInvokeStaticRange ||
			op == dex.OpInvokeInterfaceRange:
			calleeIdx := inst.VRegB_3rc()
			api := df.GetApiMethodName(calleeIdx)
			result.MethodRefs[api] = append(result.MethodRefs[api], MethodRef{
				CallerDexIdx: dexIdx,
				CallerMethod: methodIdx,
				CalleeAPI:    api,
			})

		// Instance field access: field index in VRegC_22c
		case op >= dex.OpIget && op <= dex.OpIputShort:
			fieldIdx := inst.VRegC_22c()
			api := df.GetApiFieldName(fieldIdx)
			result.FieldRefs[api] = append(result.FieldRefs[api], FieldRef{
				CallerDexIdx: dexIdx,
				CallerMethod: methodIdx,
				FieldAPI:     api,
			})

		// Static field access: field index in VRegB_21c
		case op >= dex.OpSget && op <= dex.OpSputShort:
			fieldIdx := inst.VRegB_21c()
			api := df.GetApiFieldName(fieldIdx)
			result.FieldRefs[api] = append(result.FieldRefs[api], FieldRef{
				CallerDexIdx: dexIdx,
				CallerMethod: methodIdx,
				FieldAPI:     api,
			})

		// String constants
		case op == dex.OpConstString:
			strIdx := inst.VRegB_21c()
			str := df.GetString(strIdx)
			result.StringRefs[str] = append(result.StringRefs[str], StringRef{
				CallerDexIdx: dexIdx,
				CallerMethod: methodIdx,
				Value:        str,
			})

		case op == dex.OpConstStringJumbo:
			strIdx := inst.VRegB_31c()
			str := df.GetString(strIdx)
			result.StringRefs[str] = append(result.StringRefs[str], StringRef{
				CallerDexIdx: dexIdx,
				CallerMethod: methodIdx,
				Value:        str,
			})
		}
	}
}

// ReflectAccessResult holds potential reflection access findings from cross-matching classes × strings.
type ReflectAccessResult struct {
	Signature string     // e.g., "Landroid/location/ILocationManager$Default;->getLastLocation"
	Class     string     // the class descriptor
	Member    string     // the method/field name string
	StringRef []StringRef // where the string constant was found
}

// FindPotentialReflection does veridex-style imprecise reflection detection:
// cross-match all referenced types × all string constants → check against hidden API database.
// This catches patterns like: Class.forName("android.location.ILocationManager").getMethod("getLastLocation")
//
// Optimization: instead of a full O(classes × strings) cross product, we only check strings that
// could plausibly be method/field names (no spaces, not too long, not a path), and only check
// class→member combinations that actually exist in the CSV database (by pre-indexing).
func (r *ScanResult) FindPotentialReflection(db *hiddenapi.Database) []ReflectAccessResult {
	if db == nil {
		return nil
	}

	var results []ReflectAccessResult

	// Collect candidate member name strings (from const-string in code)
	var memberStrings []stringWithRefs
	for str, refs := range r.StringRefs {
		if !isPlausibleMemberName(str) {
			continue
		}
		memberStrings = append(memberStrings, stringWithRefs{str, refs})
	}

	// Build the set of boot classes from two sources:
	// 1. type_ids referenced in DEX that are in the boot classpath
	// 2. String constants that look like Java class names, converted to DEX descriptors
	//    (veridex-style: "android.location.ILocationManager" → "Landroid/location/ILocationManager;")
	bootClassSet := make(map[string]bool)

	// Source 1: type_ids
	for cls := range r.Classes {
		if strings.HasPrefix(cls, "L") && strings.HasSuffix(cls, ";") && db.IsInBoot(cls) {
			bootClassSet[cls] = true
		}
	}

	// Source 2: string constants that look like class names (contain dots, no spaces)
	for str := range r.StringRefs {
		if strings.Contains(str, ".") && !strings.Contains(str, " ") && len(str) < 256 {
			// Convert "android.location.ILocationManager" → "Landroid/location/ILocationManager;"
			internal := "L" + strings.ReplaceAll(str, ".", "/") + ";"
			if db.IsInBoot(internal) {
				bootClassSet[internal] = true
			}
			// Also try the raw string (for JNI-style references)
			if db.IsInBoot(str) {
				bootClassSet[str] = true
			}
		}
	}

	// Build a quick lookup: member name string → StringRef
	memberLookup := make(map[string][]StringRef, len(memberStrings))
	for _, ms := range memberStrings {
		memberLookup[ms.str] = ms.refs
	}

	// Reverse approach (O(bootClasses × avgMembersPerClass) instead of O(bootClasses × allStrings)):
	// For each boot class, get its known members from the CSV database,
	// then check if any of those member names appear in our string constants.
	for cls := range bootClassSet {
		knownMembers := db.GetMembersOfClass(cls)
		if knownMembers == nil {
			continue
		}
		for memberName := range knownMembers {
			refs, found := memberLookup[memberName]
			if !found {
				continue
			}
			candidate := cls + "->" + memberName
			if db.ShouldReport(candidate) {
				results = append(results, ReflectAccessResult{
					Signature: candidate,
					Class:     cls,
					Member:    memberName,
					StringRef: refs,
				})
			}
		}
	}

	return results
}

type stringWithRefs struct {
	str  string
	refs []StringRef
}

// isPlausibleMemberName checks if a string could be a Java method or field name.
// Matching veridex's logic: a string is a potential member name only if it:
// - Does not contain spaces (not a sentence/message)
// - Does not contain '.' or '/' (those are class names, handled separately)
// - Is a valid Java identifier (starts with letter/$/_, contains only word chars)
// - Reasonable length (Java identifiers rarely exceed 128 chars)
func isPlausibleMemberName(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}

	// Must start with a Java identifier start char
	first := rune(s[0])
	if !isJavaIdentStart(first) {
		return false
	}

	for _, c := range s[1:] {
		if !isJavaIdentPart(c) {
			return false
		}
	}
	return true
}

func isJavaIdentStart(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$' || c == '<'
}

func isJavaIdentPart(c rune) bool {
	return isJavaIdentStart(c) || (c >= '0' && c <= '9') || c == '>'
}

// FilterHiddenAPIs returns only the references that match the hidden API database.
// Optimized: uses class-level pre-check to skip entire classes not in the database.
func (r *ScanResult) FilterHiddenAPIs(db *hiddenapi.Database) *ScanResult {
	if db == nil {
		return r
	}

	filtered := &ScanResult{
		MethodRefs: make(map[string][]MethodRef),
		FieldRefs:  make(map[string][]FieldRef),
		StringRefs: r.StringRefs,
		Classes:    r.Classes,
		AllStrings: r.AllStrings,
	}

	for api, refs := range r.MethodRefs {
		if db.ShouldReport(api) {
			filtered.MethodRefs[api] = refs
		}
	}

	for api, refs := range r.FieldRefs {
		if db.ShouldReport(api) {
			filtered.FieldRefs[api] = refs
		}
	}

	return filtered
}
