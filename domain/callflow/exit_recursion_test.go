package callflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

// Private body fragments isolate the fixed-point solver. Full source cases
// independently validate extraction, guards, recovery and deferred execution.
func choiceExitBodies(size, mask int) map[string]*exitBody {
	bodies := map[string]*exitBody{}
	for i := range size {
		graph := &pdg.Graph{Nodes: []pdg.Node{{Category: pdg.Return}}}
		flow := &pdg.NormalFlow{Entry: 1, Blocks: []pdg.FlowBlock{{Successors: []pdg.Ref{2}, Reachable: true}, {Operations: []pdg.Ref{1}, Reachable: true}}}
		calls := &CallSites{graph: graph, targets: map[pdg.Ref]callTargets{}}
		operations := &OperationEffects{graph: graph, orders: map[pdg.Ref]uint32{}, regions: map[pdg.Ref]uint32{}, members: map[uint32][]pdg.Ref{}}
		for j := range size {
			if mask&(1<<(i*size+j)) == 0 {
				continue
			}
			graph.Nodes = append(graph.Nodes, pdg.Node{Category: pdg.Call})
			ref := pdg.Ref(len(graph.Nodes))
			calls.targets[ref] = callTargets{known: true, names: []string{fmt.Sprint("f", j)}}
			operations.orders[ref] = uint32(ref)
			operations.regions[ref] = 1
			operations.members[1] = append(operations.members[1], ref)
			flow.Blocks = append(flow.Blocks, pdg.FlowBlock{Operations: []pdg.Ref{ref}, Successors: []pdg.Ref{2}, Reachable: true})
			flow.Blocks[0].Successors = append(flow.Blocks[0].Successors, pdg.Ref(len(flow.Blocks)))
		}
		graph.NormalFlow = flow
		bodies[fmt.Sprint("f", i)] = &exitBody{graph: graph, flow: flow, calls: calls, operations: operations}
	}
	return bodies
}
func graphMayDiverge(size, mask int) []bool {
	reachable := make([][]bool, size)
	for i := range size {
		reachable[i] = make([]bool, size)
		for j := range size {
			reachable[i][j] = mask&(1<<(i*size+j)) != 0
		}
	}
	for k := range size {
		for i := range size {
			for j := range size {
				reachable[i][j] = reachable[i][j] || reachable[i][k] && reachable[k][j]
			}
		}
	}
	result := make([]bool, size)
	for root := range size {
		for node := range size {
			result[root] = result[root] || reachable[node][node] && (root == node || reachable[root][node])
		}
	}
	return result
}
func TestExitFixedPointsAgainstIndependentInfinitePathOracle(t *testing.T) {
	checked := 0
	for size := 1; size <= 3; size++ {
		for mask := 0; mask < 1<<(size*size); mask++ {
			summaries := solveExitBodies(choiceExitBodies(size, mask))
			infinite := graphMayDiverge(size, mask)
			for i := range size {
				want := Returns
				if infinite[i] {
					want |= Diverges
				}
				got := summaries[fmt.Sprint("f", i)]
				if got.Ordinary != want || got.DeferredPanic != want {
					t.Fatalf("size=%d mask=%d root=%d got=%v want=%v", size, mask, i, got, want)
				}
				checked += 2
			}
		}
	}
	if checked != 3140 {
		t.Fatalf("checked %d contexts", checked)
	}
}
