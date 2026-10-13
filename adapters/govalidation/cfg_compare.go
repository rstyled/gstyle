package govalidation

import (
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

func firstOperations(flow *pdg.NormalFlow, starts []pdg.Ref) []pdg.Ref {
	seen := map[pdg.Ref]bool{}
	queue := slices.Clone(starts)
	var result []pdg.Ref
	for i := 0; i < len(queue); i++ {
		ref := queue[i]
		if seen[ref] {
			continue
		}
		seen[ref] = true
		block := flow.Blocks[ref-1]
		if len(block.Operations) != 0 {
			result = append(result, block.Operations[0])
			continue
		}
		queue = append(queue, block.Successors...)
	}
	return sortedRefs(result)
}

func sortedRefs(refs []pdg.Ref) []pdg.Ref {
	out := slices.Clone(refs)
	slices.Sort(out)
	return slices.Compact(out)
}

func compareSourceFlow(flow *pdg.NormalFlow, entry []pdg.Ref, expected map[pdg.Ref][]pdg.Ref) error {
	actualEntry := firstOperations(flow, []pdg.Ref{flow.Entry})
	if !slices.Equal(actualEntry, sortedRefs(entry)) {
		return fmt.Errorf("source/CFG entry correspondence disagrees")
	}
	actual := operationTransitions(flow)
	queue := slices.Clone(entry)
	seen := map[pdg.Ref]bool{}
	for i := 0; i < len(queue); i++ {
		ref := queue[i]
		if seen[ref] {
			continue
		}
		seen[ref] = true
		if err := compareOperation(actual, expected, ref); err != nil {
			return err
		}
		queue = append(queue, expected[ref]...)
	}
	if len(actual) != len(seen) {
		return fmt.Errorf("source/CFG reachable operation inventory disagrees")
	}
	return nil
}

func compareOperation(actual, expected map[pdg.Ref][]pdg.Ref, ref pdg.Ref) error {
	next, ok := actual[ref]
	if !ok {
		return fmt.Errorf("reachable source operation %d is absent from CFG", ref)
	}
	if !slices.Equal(sortedRefs(next), sortedRefs(expected[ref])) {
		return fmt.Errorf("source/CFG successors disagree at operation %d", ref)
	}
	return nil
}

func operationTransitions(flow *pdg.NormalFlow) map[pdg.Ref][]pdg.Ref {
	out := map[pdg.Ref][]pdg.Ref{}
	for _, block := range flow.Blocks {
		if !block.Reachable {
			continue
		}
		for i, ref := range block.Operations {
			out[ref] = operationNext(flow, block, i)
		}
	}
	return out
}

func operationNext(flow *pdg.NormalFlow, block pdg.FlowBlock, index int) []pdg.Ref {
	if index+1 < len(block.Operations) {
		return []pdg.Ref{block.Operations[index+1]}
	}
	return firstOperations(flow, block.Successors)
}

func compareSourceBranches(flow *pdg.NormalFlow, expected map[pdg.Ref][2][]pdg.Ref) error {
	for _, block := range flow.Blocks {
		if !block.Reachable || len(block.Operations) == 0 {
			continue
		}
		ref := block.Operations[len(block.Operations)-1]
		branches, ok := expected[ref]
		if !ok {
			continue
		}
		if err := compareBranch(flow, block, branches); err != nil {
			return err
		}
	}
	return nil
}

func compareBranch(flow *pdg.NormalFlow, block pdg.FlowBlock, expected [2][]pdg.Ref) error {
	if len(block.Successors) != 2 {
		return fmt.Errorf("source predicate requires true/false CFG successors")
	}
	for i, successor := range block.Successors {
		actual := firstOperations(flow, []pdg.Ref{successor})
		if !slices.Equal(actual, sortedRefs(expected[i])) {
			return fmt.Errorf("source/CFG branch outcome correspondence disagrees")
		}
	}
	return nil
}
