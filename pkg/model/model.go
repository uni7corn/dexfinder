// Package model defines the structured output model for dexfinder results.
// This model provides a rich, type-safe representation that can be serialized
// to JSON, consumed by IDE plugins, CI pipelines, or web UIs.
package model

// AnalysisResult is the top-level structured output of a dexfinder scan.
type AnalysisResult struct {
	// Metadata about the analyzed APK/DEX
	Metadata Metadata `json:"metadata"`

	// All classes defined in the APK
	Classes []ClassInfo `json:"classes,omitempty"`

	// Method call references found
	MethodCalls []MethodCallInfo `json:"method_calls,omitempty"`

	// Field access references found
	FieldAccesses []FieldAccessInfo `json:"field_accesses,omitempty"`

	// String constant references found in code
	StringRefs []StringRefInfo `json:"string_refs,omitempty"`

	// Hidden API findings (if --api-flags provided)
	HiddenAPIs []HiddenAPIFinding `json:"hidden_apis,omitempty"`

	// Reflection findings (imprecise cross-match)
	ReflectionFindings []ReflectionFinding `json:"reflection_findings,omitempty"`

	// Call chains (if --trace provided)
	CallChains []CallChainInfo `json:"call_chains,omitempty"`

	// Summary statistics
	Summary Summary `json:"summary"`
}

// Metadata describes the analyzed input.
type Metadata struct {
	FilePath    string `json:"file_path"`
	FileSize    int64  `json:"file_size"`
	DexCount    int    `json:"dex_count"`
	Query       string `json:"query,omitempty"`
	MappingFile string `json:"mapping_file,omitempty"`
	FlagsFile   string `json:"flags_file,omitempty"`
}

// ClassInfo describes a class found in the DEX.
type ClassInfo struct {
	// DEX descriptor: "Lcom/example/Foo;"
	Descriptor string `json:"descriptor"`

	// Java-style name: "com.example.Foo"
	JavaName string `json:"java_name"`

	// Original name if deobfuscated via mapping
	OriginalName string `json:"original_name,omitempty"`

	// Access flags
	AccessFlags uint32 `json:"access_flags"`

	// Superclass descriptor
	Superclass string `json:"superclass,omitempty"`

	// Interface descriptors
	Interfaces []string `json:"interfaces,omitempty"`

	// Number of methods and fields
	MethodCount int `json:"method_count"`
	FieldCount  int `json:"field_count"`

	// DEX file index (for multi-dex)
	DexIndex int `json:"dex_index"`
}

// MethodInfo identifies a method.
type MethodInfo struct {
	// Full DEX signature: "Lcom/example/Foo;->bar(Ljava/lang/String;)V"
	DexSignature string `json:"dex_signature"`

	// Class descriptor
	Class string `json:"class"`

	// Method name
	Name string `json:"name"`

	// Parameter types in DEX format
	ParamTypes []string `json:"param_types,omitempty"`

	// Return type in DEX format
	ReturnType string `json:"return_type"`

	// Java-style readable: "com.example.Foo.bar(String)"
	JavaReadable string `json:"java_readable"`

	// Original name if deobfuscated
	OriginalSignature string `json:"original_signature,omitempty"`

	// DEX file index
	DexIndex int `json:"dex_index"`

	// Method index within the DEX file
	MethodIndex uint32 `json:"method_index"`
}

// FieldInfo identifies a field.
type FieldInfo struct {
	// Full DEX signature: "Lcom/example/Foo;->bar:I"
	DexSignature string `json:"dex_signature"`

	// Class descriptor
	Class string `json:"class"`

	// Field name
	Name string `json:"name"`

	// Field type descriptor
	Type string `json:"type"`

	// Original name if deobfuscated
	OriginalSignature string `json:"original_signature,omitempty"`

	DexIndex   int    `json:"dex_index"`
	FieldIndex uint32 `json:"field_index"`
}

// Location represents where something was found in the code.
type Location struct {
	// The method containing this reference
	Method MethodInfo `json:"method"`

	// Byte offset within the method's instructions (DexPC)
	DexPC uint32 `json:"dex_pc,omitempty"`

	// Source file name (from DEX debug info or mapping)
	SourceFile string `json:"source_file,omitempty"`

	// Source line number (if available from mapping)
	LineNumber int `json:"line_number,omitempty"`
}

// MethodCallInfo represents a method invocation found in the bytecode.
type MethodCallInfo struct {
	// The method being called
	Target MethodInfo `json:"target"`

	// Where this call occurs
	Locations []Location `json:"locations"`

	// Number of call sites
	Count int `json:"count"`

	// Invoke type: "virtual", "static", "direct", "interface", "super"
	InvokeType string `json:"invoke_type,omitempty"`
}

// FieldAccessInfo represents a field access found in the bytecode.
type FieldAccessInfo struct {
	// The field being accessed
	Target FieldInfo `json:"target"`

	// Where this access occurs
	Locations []Location `json:"locations"`

	// Number of access sites
	Count int `json:"count"`

	// Access type: "iget", "iput", "sget", "sput"
	AccessType string `json:"access_type,omitempty"`
}

// StringRefInfo represents a string constant used in code.
type StringRefInfo struct {
	// The string value
	Value string `json:"value"`

	// Where this string is used
	Locations []Location `json:"locations"`

	Count int `json:"count"`

	// Whether this string was only found in the DEX string table (no code reference)
	TableOnly bool `json:"table_only,omitempty"`
}

// HiddenAPIFinding represents a hidden API usage.
type HiddenAPIFinding struct {
	// The hidden API signature
	Signature string `json:"signature"`

	// API restriction level: "blocked", "unsupported", "max-target-o", etc.
	Restriction string `json:"restriction"`

	// How it was found: "linking" or "reflection"
	AccessType string `json:"access_type"`

	// Where it's used
	Locations []Location `json:"locations"`
}

// ReflectionFinding represents a potential reflection-based hidden API access.
type ReflectionFinding struct {
	// Reconstructed signature: "Landroid/location/ILocationManager;->getLastLocation"
	Signature string `json:"signature"`

	// The class being reflected on
	TargetClass string `json:"target_class"`

	// The member name being accessed
	MemberName string `json:"member_name"`

	// API restriction level
	Restriction string `json:"restriction"`

	// Where the member name string constant appears
	StringLocations []Location `json:"string_locations"`
}

// CallChainInfo represents one complete call chain from root to target.
type CallChainInfo struct {
	// The target API this chain leads to
	Target string `json:"target"`

	// The chain of methods, from outermost caller to target
	// Index 0 = root caller, last = target
	Chain []CallChainEntry `json:"chain"`

	// Chain depth
	Depth int `json:"depth"`
}

// CallChainEntry is one frame in a call chain.
type CallChainEntry struct {
	Method MethodInfo `json:"method"`

	// True if this entry is a recursive call (cycle detected)
	IsCycle bool `json:"is_cycle,omitempty"`
}

// Summary provides aggregate statistics.
type Summary struct {
	TotalClasses     int `json:"total_classes"`
	TotalMethods     int `json:"total_methods"`
	TotalFields      int `json:"total_fields"`
	TotalStrings     int `json:"total_strings"`
	TotalMethodCalls int `json:"total_method_calls"`
	TotalFieldAccess int `json:"total_field_accesses"`
	HiddenAPICount   int `json:"hidden_api_count"`
	LinkingCount     int `json:"linking_count"`
	ReflectionCount  int `json:"reflection_count"`
	CallChainsCount  int `json:"call_chains_count"`
}
