package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

func (evaluation *exitEvaluation) transferCall(point exitPoint, ref pdg.Ref) (exitPoint, bool) {
	body := evaluation.body
	if body.graph.Nodes[ref-1].Category != pdg.Call {
		return point, false
	}
	if body.calls.recoveryCall(ref) {
		return point, true
	}
	invocation := exitInvocationContext{ref: ref, values: evaluation.valueContext(point)}
	projection := body.calls.projectEnvironment(evaluation.summaries, invocation, Returns)
	proof := body.calls.invocationMemory(evaluation.summaries, invocation)
	if !proof.preserves {
		point = evaluation.unorderedReadBarrier(point, ref)
	}
	point = evaluation.applyEnvironmentProjection(point, invocationCompletion{invocation, projection, proof, body.operations.orderedRegion(ref)})
	return evaluation.storeCallResult(point, ref, projection.returned), true
}

func (evaluation *exitEvaluation) snapshotRead(point exitPoint, ref pdg.Ref) exitPoint {
	body := evaluation.body
	if body.graph.Nodes[ref-1].Category != pdg.Read || !evaluation.regionHasWriter(ref) {
		return point
	}
	values := evaluation.values[point.values]
	value := unknownValue
	if !values.unorderedReads[ref] {
		context := evaluation.valueContext(point)
		context.Nodes = maps.Clone(context.Nodes)
		delete(context.Nodes, ref)
		value = abstractValue(body.paths.Evaluate(ref, context))
	}
	next := values
	next.nodes = maps.Clone(values.nodes)
	if next.nodes == nil {
		next.nodes = map[pdg.Ref]exitValue{}
	}
	next.nodes[ref] = value
	point.values = evaluation.internValues(next)
	return point
}

func (evaluation *exitEvaluation) regionHasWriter(ref pdg.Ref) bool {
	operations := evaluation.body.operations
	if !operations.SameRegion(ref, ref) {
		return false
	}
	for _, candidate := range operations.members[operations.regions[ref]] {
		if evaluation.body.graph.Nodes[candidate-1].Category != pdg.Call {
			continue
		}
		if !evaluation.body.calls.PreservesCallerMemory(evaluation.summaries, candidate) {
			return true
		}
	}
	return false
}

func (evaluation *exitEvaluation) unorderedReadBarrier(point exitPoint, ref pdg.Ref) exitPoint {
	operations := evaluation.body.operations
	if !operations.orderedRegion(ref) {
		return point
	}
	values := evaluation.values[point.values]
	next := values
	next.nodes = cloneValueMap(values.nodes)
	next.unorderedReads = cloneReadBarriers(values.unorderedReads)
	for _, candidate := range operations.members[operations.regions[ref]] {
		if !operations.unorderedRead(candidate, ref) {
			continue
		}
		next.nodes[candidate] = unknownValue
		next.unorderedReads[candidate] = true
	}
	point.values = evaluation.internValues(next)
	return point
}

func (operations *OperationEffects) unorderedRead(candidate, call pdg.Ref) bool {
	if operations.graph.Nodes[candidate-1].Category != pdg.Read {
		return false
	}
	group := operations.regions[call]
	return !operations.mustBefore(candidate, call, group) && !operations.mustAfter(candidate, call, group)
}

func cloneValueMap(values map[pdg.Ref]exitValue) map[pdg.Ref]exitValue {
	result := map[pdg.Ref]exitValue{}
	maps.Copy(result, values)
	return result
}

func cloneReadBarriers(values map[pdg.Ref]bool) map[pdg.Ref]bool {
	result := map[pdg.Ref]bool{}
	maps.Copy(result, values)
	return result
}
