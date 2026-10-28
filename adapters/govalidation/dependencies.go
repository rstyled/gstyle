package govalidation

import (
	"fmt"

	"github.com/rstyled/gstyle/domain/pdg"
)

type dependency struct {
	from, to pdg.Ref
	kind     pdg.EdgeKind
	position pdg.Position
	outcome  string
}

type controlArm struct {
	controller pdg.Ref
	position   pdg.Position
}

// Ordering independently checks the complete sparse execution/control basis on
// the validated acyclic operation CFG. Scratch storage is linear in V+E.
func Ordering(g *pdg.Graph) error {
	if err := pdg.Validate(g); err != nil {
		return err
	}
	if g.NormalFlow == nil {
		return fmt.Errorf("normal-flow context is missing")
	}
	next := operationTransitions(g.NormalFlow)
	next[0] = firstOperations(g.NormalFlow, []pdg.Ref{g.NormalFlow.Entry})
	dom, err := dominanceTree(next, 0)
	if err != nil {
		return err
	}
	expected := executionDependencies(dom)
	post, err := dominanceTree(reverseFlow(next), 0)
	if err != nil {
		return err
	}
	controlFlowDependencies(g.NormalFlow, expected, post)
	return compareOrdering(g, expected)
}

func executionDependencies(dom map[pdg.Ref]pdg.Ref) map[dependency]bool {
	expected := map[dependency]bool{}
	for ref, parent := range dom {
		if ref != 0 && parent != 0 {
			expected[dependency{from: parent, to: ref, kind: pdg.Execution}] = true
		}
	}
	return expected
}

func reverseFlow(next map[pdg.Ref][]pdg.Ref) map[pdg.Ref][]pdg.Ref {
	out := map[pdg.Ref][]pdg.Ref{0: nil}
	for from, successors := range next {
		if from == 0 {
			continue
		}
		if len(successors) == 0 {
			out[0] = append(out[0], from)
		}
		if _, ok := out[from]; !ok {
			out[from] = nil
		}
		for _, to := range successors {
			out[to] = append(out[to], from)
		}
	}
	return out
}

func controlFlowDependencies(flow *pdg.NormalFlow, expected map[dependency]bool, post map[pdg.Ref]pdg.Ref) {
	for _, block := range flow.Blocks {
		if !block.Reachable || len(block.Operations) == 0 || len(block.Successors) != 2 {
			continue
		}
		controller := block.Operations[len(block.Operations)-1]
		for arm, successor := range block.Successors {
			branch := controlArm{controller, pdg.Position(arm + 1)}
			controlDependencies(expected, branch, firstOperations(flow, []pdg.Ref{successor}), post)
		}
	}
}

func controlDependencies(expected map[dependency]bool, arm controlArm, successors []pdg.Ref, post map[pdg.Ref]pdg.Ref) {
	if arm.controller == 0 {
		return
	}
	stop := post[arm.controller]
	for _, next := range successors {
		for runner := next; runner != stop && runner != 0; runner = post[runner] {
			if arm.controller != runner {
				expected[dependency{arm.controller, runner, pdg.ControlEdge, arm.position, [2]string{"true", "false"}[arm.position-1]}] = true
			}
		}
	}
}

func compareOrdering(g *pdg.Graph, expected map[dependency]bool) error {
	for _, edge := range g.Edges {
		if edge.Kind == pdg.Data {
			continue
		}
		key := dependency{edge.Source, edge.Target, edge.Kind, edge.Position, g.Tables.Text(edge.Outcome)}
		if !expected[key] {
			return fmt.Errorf("unexpected or duplicate ordering dependency %d -> %d", edge.Source, edge.Target)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		return fmt.Errorf("missing %d ordering dependencies", len(expected))
	}
	return nil
}

func dominanceTree(next map[pdg.Ref][]pdg.Ref, entry pdg.Ref) (map[pdg.Ref]pdg.Ref, error) {
	predecessors, remaining := predecessorCounts(next)
	queue := []pdg.Ref{entry}
	parents := map[pdg.Ref]pdg.Ref{entry: entry}
	depth := map[pdg.Ref]int{entry: 0}
	for i := 0; i < len(queue); i++ {
		current := queue[i]
		if current != entry {
			parent := commonAncestor(predecessors[current], parents, depth)
			parents[current] = parent
			depth[current] = depth[parent] + 1
		}
		for _, to := range sortedRefs(next[current]) {
			remaining[to]--
			if remaining[to] == 0 {
				queue = append(queue, to)
			}
		}
	}
	if len(parents) != len(next) {
		return nil, fmt.Errorf("ordering CFG is cyclic or has an unconnected operation")
	}
	return parents, nil
}

func predecessorCounts(next map[pdg.Ref][]pdg.Ref) (map[pdg.Ref][]pdg.Ref, map[pdg.Ref]int) {
	predecessors := map[pdg.Ref][]pdg.Ref{}
	remaining := map[pdg.Ref]int{}
	for from, successors := range next {
		if _, ok := remaining[from]; !ok {
			remaining[from] = 0
		}
		for _, to := range sortedRefs(successors) {
			predecessors[to] = append(predecessors[to], from)
			remaining[to]++
		}
	}
	return predecessors, remaining
}

func commonAncestor(predecessors []pdg.Ref, parents map[pdg.Ref]pdg.Ref, depth map[pdg.Ref]int) pdg.Ref {
	common := predecessors[0]
	for _, other := range predecessors[1:] {
		common = ancestorIntersection(common, other, parents, depth)
	}
	return common
}

func ancestorIntersection(left, right pdg.Ref, parents map[pdg.Ref]pdg.Ref, depth map[pdg.Ref]int) pdg.Ref {
	for left != right {
		if depth[left] > depth[right] {
			left = parents[left]
			continue
		}
		right = parents[right]
	}
	return left
}
