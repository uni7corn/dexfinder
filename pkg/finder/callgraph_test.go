package finder

import (
	"testing"
)

// buildMockCallGraph creates a call graph from string pairs for testing.
// pairs: [callee, caller, callee, caller, ...]
func buildMockCallGraph(pairs ...string) *CallGraph {
	cg := &CallGraph{
		callers:  make(map[string][]MethodID),
		callees:  make(map[MethodID][]string),
		dexFiles: nil, // not needed for mock
	}
	// Override ResolveMethodName to return MethodID as string directly
	// We'll use a trick: store method names in callers map with dummy MethodIDs
	nameMap := make(map[MethodID]string)
	idx := uint32(0)
	for i := 0; i < len(pairs)-1; i += 2 {
		callee := pairs[i]
		caller := pairs[i+1]
		mid := MethodID{DexIdx: 0, MethodIdx: idx}
		idx++
		cg.callers[callee] = append(cg.callers[callee], mid)
		nameMap[mid] = caller
	}
	// Patch ResolveMethodName via a wrapper
	cg.nameResolver = nameMap
	return cg
}

func TestTraceCallers_Simple(t *testing.T) {
	cg := buildMockCallGraph(
		"target", "caller1",
		"target", "caller2",
		"caller1", "root",
	)

	tree := cg.TraceCallers("target", 5)
	if tree == nil {
		t.Fatal("tree is nil")
	}
	if len(tree.Callers) != 2 {
		t.Fatalf("expected 2 callers, got %d", len(tree.Callers))
	}

	// caller1 should have root as its caller
	var caller1 *CallChainNode
	for _, c := range tree.Callers {
		if c.Method == "caller1" {
			caller1 = c
		}
	}
	if caller1 == nil {
		t.Fatal("caller1 not found")
	}
	if len(caller1.Callers) != 1 || caller1.Callers[0].Method != "root" {
		t.Error("caller1 should have root as caller")
	}
}

func TestTraceCallers_DirectRecursion(t *testing.T) {
	// A calls itself: A → A
	cg := buildMockCallGraph(
		"target", "A",
		"A", "A", // self-recursion
		"A", "B",
	)

	tree := cg.TraceCallers("target", 10)
	if tree == nil {
		t.Fatal("tree is nil")
	}

	// Should find A as caller of target
	if len(tree.Callers) != 1 || tree.Callers[0].Method != "A" {
		t.Fatalf("expected A as caller, got %v", tree.Callers)
	}

	nodeA := tree.Callers[0]
	// A's callers should include A(cycle) and B
	foundCycle := false
	foundB := false
	for _, c := range nodeA.Callers {
		if c.Method == "A" && c.IsCycle {
			foundCycle = true
		}
		if c.Method == "B" {
			foundB = true
		}
	}
	if !foundCycle {
		t.Error("expected self-recursive A to be marked as cycle")
	}
	if !foundB {
		t.Error("expected B as caller of A")
	}
}

func TestTraceCallers_MutualRecursion(t *testing.T) {
	// A → B → A (mutual recursion)
	cg := buildMockCallGraph(
		"target", "A",
		"A", "B",
		"B", "A",
	)

	tree := cg.TraceCallers("target", 10)

	// target → A → B → A(cycle)
	nodeA := tree.Callers[0]
	if nodeA.Method != "A" {
		t.Fatalf("expected A, got %s", nodeA.Method)
	}
	if len(nodeA.Callers) != 1 || nodeA.Callers[0].Method != "B" {
		t.Fatal("expected B as caller of A")
	}
	nodeB := nodeA.Callers[0]
	if len(nodeB.Callers) != 1 || nodeB.Callers[0].Method != "A" || !nodeB.Callers[0].IsCycle {
		t.Error("expected A(cycle) as caller of B")
	}
}

func TestTraceCallers_DiamondPath(t *testing.T) {
	// Diamond: target ← A ← C, target ← B ← C
	// C should appear in both branches (not skipped due to global visited)
	cg := buildMockCallGraph(
		"target", "A",
		"target", "B",
		"A", "C",
		"B", "C",
	)

	tree := cg.TraceCallers("target", 5)
	if len(tree.Callers) != 2 {
		t.Fatalf("expected 2 callers (A, B), got %d", len(tree.Callers))
	}

	// Both A and B should have C as caller
	for _, caller := range tree.Callers {
		if len(caller.Callers) != 1 || caller.Callers[0].Method != "C" {
			t.Errorf("caller %s should have C, got %v", caller.Method, caller.Callers)
		}
		if caller.Callers[0].IsCycle {
			t.Errorf("C under %s should NOT be marked as cycle", caller.Method)
		}
	}
}

func TestTraceCallers_DepthLimit(t *testing.T) {
	// Long chain: target ← A ← B ← C ← D ← E
	cg := buildMockCallGraph(
		"target", "A",
		"A", "B",
		"B", "C",
		"C", "D",
		"D", "E",
	)

	tree := cg.TraceCallers("target", 3)
	// Should only go 3 levels deep: A → B → C, no D or E
	nodeA := tree.Callers[0]
	nodeB := nodeA.Callers[0]
	nodeC := nodeB.Callers[0]
	if len(nodeC.Callers) != 0 {
		t.Error("depth limit should stop at level 3")
	}
}

func TestFlatCallerChains_WithCycle(t *testing.T) {
	cg := buildMockCallGraph(
		"target", "A",
		"A", "B",
		"B", "A", // cycle
	)

	tree := cg.TraceCallers("target", 10)
	chains := FlatCallerChains(tree, 50)

	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	chain := chains[0]
	// Should be: target, A, B, A [recursive]
	if len(chain) != 4 {
		t.Fatalf("expected chain length 4, got %d: %v", len(chain), chain)
	}
	lastEntry := chain[len(chain)-1]
	if lastEntry != "A [recursive]" {
		t.Errorf("last entry should be 'A [recursive]', got %q", lastEntry)
	}
}
