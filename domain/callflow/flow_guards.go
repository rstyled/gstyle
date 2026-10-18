package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type flowGuard struct {
	condition pdg.Ref
	truth     bool
}

func (operations *OperationEffects) indexFlowGuards() {
	flow := pdg.ExitBodyFlow(operations.graph)
	if flow == nil {
		return
	}
	for i, block := range flow.Blocks {
		if !block.Reachable || block.Condition == 0 || len(block.Successors) != 2 {
			continue
		}
		operations.indexFlowGuard(flow, pdg.Ref(i+1), block)
	}
}

func (operations *OperationEffects) indexFlowGuard(flow *pdg.NormalFlow, ref pdg.Ref, guard pdg.FlowBlock) {
	outside := reachableAvoiding(flow, flow.Entry, ref)
	left := reachableAvoiding(flow, guard.Successors[0], ref)
	right := reachableAvoiding(flow, guard.Successors[1], ref)
	for i, block := range flow.Blocks {
		if outside[i] || left[i] == right[i] {
			continue
		}
		operations.addFlowGuard(block.Operations, flowGuard{condition: guard.Condition, truth: left[i]})
	}
}

func (operations *OperationEffects) addFlowGuard(refs []pdg.Ref, guard flowGuard) {
	for _, ref := range refs {
		operations.guards[ref] = append(operations.guards[ref], guard)
	}
}

func (operations *OperationEffects) guardDependsOn(ref, target pdg.Ref, group uint32, seen map[pdg.Ref]bool) bool {
	for _, guard := range operations.guards[ref] {
		if operations.dependsOn(guard.condition, target, group, seen) {
			return true
		}
	}
	return false
}

func (operations *OperationEffects) allowsExecution(ref pdg.Ref, context bindingflow.Evaluation) bool {
	for _, guard := range operations.guards[ref] {
		if !operations.SameRegion(ref, guard.condition) {
			continue
		}
		truth, known := operations.paths.Evaluate(guard.condition, context).Bool()
		if known && truth != guard.truth {
			return false
		}
	}
	return true
}

// A guard must dominate the candidate and only one arm may reach it before
// testing that guard again. A sibling after the join is not guarded work.
func reachableAvoiding(flow *pdg.NormalFlow, start, blocked pdg.Ref) []bool {
	seen := make([]bool, len(flow.Blocks))
	queue := []pdg.Ref{start}
	for at := 0; at < len(queue); at++ {
		ref := queue[at]
		if ref == blocked || seen[ref-1] {
			continue
		}
		seen[ref-1] = true
		queue = append(queue, flow.Blocks[ref-1].Successors...)
	}
	return seen
}
