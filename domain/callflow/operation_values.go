package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

// Current bindings cannot settle a sibling fault when calls in the same
// evaluation region may change those bindings in another permitted order.
func (operations *OperationEffects) stableInputRegion(ref pdg.Ref) bool {
	if !operations.SameRegion(ref, ref) {
		return false
	}
	members := operations.members[operations.regions[ref]]
	if !slices.Contains(members, ref) {
		return false
	}
	for _, member := range members {
		if operations.graph.Nodes[member-1].Category == pdg.Call {
			return false
		}
	}
	return true
}

func (operations *OperationEffects) outcomesWithValues(ref pdg.Ref, context bindingflow.Evaluation) Outcomes {
	outcomes := operations.Outcomes(ref)
	if outcomes&Panics == 0 {
		return outcomes
	}
	effect := operations.effect(ref)
	if !effect.NilInput || effect.Unknown {
		return outcomes
	}
	if !operations.stableInputRegion(ref) {
		return outcomes
	}
	input := operations.paths.Evaluate(operations.paths.Operand(ref, 1), context)
	return nilInputOutcomes(effect, input)
}

func nilInputOutcomes(effect operationEffect, input bindingflow.Value) Outcomes {
	if input.Nil {
		return Panics
	}
	if input.NonNil {
		effect.NilInput = false
	}
	return effectOutcomes(effect)
}
