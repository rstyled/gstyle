package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

type invocationCompletion struct {
	invocation exitInvocationContext
	projection environmentProjection
	proof      memoryProof
	ordered    bool
}

func (evaluation *exitEvaluation) applyEnvironmentProjection(point exitPoint, completion invocationCompletion) exitPoint {
	invocation, projection, proof, ordered := completion.invocation, completion.projection, completion.proof, completion.ordered
	exact := projection.known && proof.confined && ordered
	if !proof.preserves && !exact {
		point = evaluation.havocEnvironment(point, invocation)
	}
	if exact {
		point = evaluation.captureWriteback(point, projection.writes)
	}
	return evaluation.installEnvironment(point, projection, proof, exact)
}
func (evaluation *exitEvaluation) havocEnvironment(point exitPoint, invocation exitInvocationContext) exitPoint {
	point = evaluation.rawValueTransfer(point, invocation.ref)
	values := evaluation.values[point.values]
	values.heap = unknownHeap(values.heap)
	point.values = evaluation.internValues(values)
	return point
}
func (evaluation *exitEvaluation) installEnvironment(point exitPoint, projection environmentProjection, proof memoryProof, exact bool) exitPoint {
	values := evaluation.values[point.values]
	old := values.heap
	values.heap = maps.Clone(values.heap)
	if values.heap == nil {
		values.heap = environmentHeap{}
	}
	values.heap.install(projection, proof, exact)
	values.bindings = values.changedEnvironmentBindings(old)
	point.values = evaluation.internValues(values)
	return point
}
func (heap environmentHeap) install(projection environmentProjection, proof memoryProof, exact bool) {
	for address, cell := range projection.heap {
		fresh := projection.fresh[address]
		if !fresh && (proof.preserves || !exact) {
			continue
		}
		cell = heap.importedCell(address, cell, fresh)
		if !projection.known {
			cell.Value = unknownValue
		}
		heap[address] = cell
	}
}
func (heap environmentHeap) importedCell(address string, cell environmentCell, fresh bool) environmentCell {
	if previous, found := heap[address]; found && fresh {
		cell.Value = joinValue(previous.Value, cell.Value)
		cell.Repeated = true
	}
	return cell
}
func (values exitValues) changedEnvironmentBindings(old environmentHeap) map[pdg.Ref]exitValue {
	bindings := cloneValueMap(values.bindings)
	for symbol, address := range values.locations {
		cell := values.heap[address]
		if cell == old[address] {
			continue
		}
		if cell.Repeated {
			delete(bindings, symbol)
		} else {
			bindings[symbol] = cell.Value
		}
	}
	return bindings
}
