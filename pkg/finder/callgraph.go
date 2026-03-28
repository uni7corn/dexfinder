package finder

import (
	"dex_method_finder/pkg/dex"
)

// MethodID uniquely identifies a method across DEX files.
type MethodID struct {
	DexIdx    int
	MethodIdx uint32
}

// CallGraph represents the method call relationships in the app.
type CallGraph struct {
	// callers: callee API string → list of caller MethodIDs
	callers map[string][]MethodID
	// callees: caller MethodID → list of callee API strings
	callees map[MethodID][]string
	// dexFiles for resolving method names
	dexFiles []*dex.DexFile
	// nameResolver overrides dexFiles resolution (for testing)
	nameResolver map[MethodID]string
}

// CallChainNode represents one node in a call chain.
type CallChainNode struct {
	Method   string           // full API name
	Callers  []*CallChainNode // who calls this method
	Depth    int
	IsCycle  bool             // true if this node was already on the current path (recursive call)
}

// BuildCallGraph constructs the call graph from scan results.
func BuildCallGraph(result *ScanResult, dexFiles []*dex.DexFile) *CallGraph {
	cg := &CallGraph{
		callers:  make(map[string][]MethodID),
		callees:  make(map[MethodID][]string),
		dexFiles: dexFiles,
	}

	// Build from method references
	for api, refs := range result.MethodRefs {
		for _, ref := range refs {
			mid := MethodID{DexIdx: ref.CallerDexIdx, MethodIdx: ref.CallerMethod}
			cg.callers[api] = append(cg.callers[api], mid)
			cg.callees[mid] = append(cg.callees[mid], api)
		}
	}

	return cg
}

// ResolveMethodName returns the full API name for a MethodID.
func (cg *CallGraph) ResolveMethodName(mid MethodID) string {
	if cg.nameResolver != nil {
		return cg.nameResolver[mid]
	}
	if mid.DexIdx < len(cg.dexFiles) {
		return cg.dexFiles[mid.DexIdx].GetApiMethodName(mid.MethodIdx)
	}
	return ""
}

// TraceCallers returns the call chain leading to the target API, up to maxDepth levels.
// Returns a tree of callers. Handles recursive/cyclic calls safely using per-path visited tracking.
func (cg *CallGraph) TraceCallers(targetAPI string, maxDepth int) *CallChainNode {
	root := &CallChainNode{
		Method: targetAPI,
		Depth:  0,
	}

	// pathVisited tracks the current path from root to the node being expanded,
	// preventing cycles within a single path while allowing the same method
	// to appear in different branches of the tree.
	pathVisited := make(map[string]bool)
	pathVisited[targetAPI] = true

	cg.traceCallersRecursive(root, maxDepth, pathVisited)
	return root
}

func (cg *CallGraph) traceCallersRecursive(node *CallChainNode, maxDepth int, pathVisited map[string]bool) {
	if node.Depth >= maxDepth {
		return
	}

	callerIDs := cg.callers[node.Method]
	if len(callerIDs) == 0 {
		return
	}

	// Deduplicate callers at the same level by resolved name
	seen := make(map[string]bool)
	for _, mid := range callerIDs {
		callerName := cg.ResolveMethodName(mid)
		if callerName == "" || seen[callerName] {
			continue
		}
		seen[callerName] = true

		child := &CallChainNode{
			Method: callerName,
			Depth:  node.Depth + 1,
		}

		if pathVisited[callerName] {
			// Cycle detected on the current path — add the node but mark it
			// and don't recurse further. This makes the cycle visible in output.
			child.IsCycle = true
			node.Callers = append(node.Callers, child)
			continue
		}

		node.Callers = append(node.Callers, child)

		// Push onto path, recurse, then pop (backtrack)
		pathVisited[callerName] = true
		cg.traceCallersRecursive(child, maxDepth, pathVisited)
		delete(pathVisited, callerName)
	}
}

// FlatCallerChains returns all caller chains as flat string slices, for easier output.
// Each chain is from the target (index 0) up to the root caller (last index).
// Cycle nodes are treated as leaf nodes (the chain ends there with a "[cycle]" marker).
func FlatCallerChains(root *CallChainNode, maxChains int) [][]string {
	var chains [][]string
	var dfs func(node *CallChainNode, path []string)
	dfs = func(node *CallChainNode, path []string) {
		if len(chains) >= maxChains {
			return
		}
		label := node.Method
		if node.IsCycle {
			label = node.Method + " [recursive]"
		}
		current := make([]string, len(path)+1)
		copy(current, path)
		current[len(path)] = label

		// Leaf or cycle node: emit chain
		if len(node.Callers) == 0 || node.IsCycle {
			chain := make([]string, len(current))
			copy(chain, current)
			chains = append(chains, chain)
			return
		}
		for _, caller := range node.Callers {
			dfs(caller, current)
		}
	}
	dfs(root, nil)
	return chains
}

// GetDirectCallers returns the deduplicated direct callers of an API.
func (cg *CallGraph) GetDirectCallers(api string) []string {
	callerIDs := cg.callers[api]
	seen := make(map[string]bool)
	var result []string
	for _, mid := range callerIDs {
		name := cg.ResolveMethodName(mid)
		if name != "" && !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}
