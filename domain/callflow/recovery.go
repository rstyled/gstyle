package callflow

import "github.com/rstyled/gstyle/domain/pdg"

func (evaluation *exitEvaluation) recovery(point exitPoint, ref pdg.Ref) Outcomes {
	valuePoint := evaluation.recoveryValue(point, ref)
	next := exitPoint{
		block: point.block, offset: point.offset, stack: point.stack,
		values: valuePoint.values, recovery: ref,
		recovered: point.recovered || point.pending, pending: false, deferred: point.deferred,
	}
	return evaluation.visit(evaluation.transferValues(next, ref)) | evaluation.postRecoveryEffects(next, ref)
}

// Non-call sibling faults can follow recover even when the compiler's selected
// traversal puts them first. PriorOutcomes already retains the reverse order.
// Call invocations keep their canonical language order and are not moved here.
func (evaluation *exitEvaluation) postRecoveryEffects(point exitPoint, ref pdg.Ref) Outcomes {
	candidates, known := evaluation.body.operations.LaterOperations(ref)
	if !known {
		return UnknownExit
	}
	context := evaluation.valueContext(point)
	var result Outcomes
	for _, candidate := range candidates {
		if evaluation.body.graph.Nodes[candidate-1].Category == pdg.Call || !evaluation.body.operations.allowsExecution(candidate, context) {
			continue
		}
		result |= evaluation.finish(point, evaluation.body.operations.Outcomes(candidate)&^Returns)
	}
	return result
}

func (evaluation *exitEvaluation) mayFollowRecovery(point exitPoint, candidate pdg.Ref) bool {
	operations := evaluation.body.operations
	if point.recovery == 0 || !operations.SameRegion(candidate, point.recovery) {
		return true
	}
	return !operations.mustBefore(candidate, point.recovery, operations.regions[point.recovery])
}
