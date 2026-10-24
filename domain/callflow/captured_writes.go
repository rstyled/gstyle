package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

type captureProjection struct {
	invocation     exitInvocationContext
	outcomes, mode Outcomes
}

func (calls *CallSites) targetCaptureModeWrites(name string, summaries exitSummaryReader, projection captureProjection) map[pdg.Ref]exitValue {
	invocation := projection.invocation
	summary, _ := readExitSummary(summaries, name)
	result := map[pdg.Ref]exitValue{}
	if calls.targetInvocationMemory(name, summaries, invocation).preserves || summary.captureArity == 0 {
		return result
	}
	values := calls.projectedCaptureValues(summary, projection)
	if len(invocation.target.Cells) != 0 {
		return result
	}
	links := summary.invocations.captureBindings(calls.graph)
	for i, symbol := range summary.invocations.captures {
		result[links[symbol]] = values[i]
	}
	return result
}

func joinCaptureWrites(alternatives []map[pdg.Ref]exitValue, bindings map[pdg.Ref]bindingflow.Value) map[pdg.Ref]exitValue {
	result := map[pdg.Ref]exitValue{}
	for _, writes := range alternatives {
		for symbol := range writes {
			result[symbol] = joinedCaptureValue(alternatives, symbol, abstractValue(bindings[symbol]))
		}
	}
	return result
}

func joinedCaptureValue(alternatives []map[pdg.Ref]exitValue, symbol pdg.Ref, initial exitValue) exitValue {
	var value exitValue
	for i, writes := range alternatives {
		next, found := writes[symbol]
		if !found {
			next = initial
		}
		if i != 0 && next != value {
			return unknownValue
		}
		value = next
	}
	return value
}

func (evaluation *exitEvaluation) captureWriteback(point exitPoint, writes map[pdg.Ref]exitValue) exitPoint {
	old := evaluation.values[point.values]
	next := old
	next.bindings = maps.Clone(old.liveBindings())
	next.cells = maps.Clone(old.cells)
	next.heap = maps.Clone(old.heap)
	if next.cells == nil {
		next.cells = map[pdg.Ref]exitValue{}
	}
	if next.bindings == nil {
		next.bindings = map[pdg.Ref]exitValue{}
	}
	for symbol, value := range writes {
		next.cells[symbol] = joinCellValue(next.cells, symbol, value)
		if address := next.locations[symbol]; address != "" {
			writeEnvironmentCell(next.heap, address, value)
		}
		delete(next.bindings, symbol)
		if evaluation.body.facts.SingleCellSymbols[symbol] {
			next.bindings[symbol] = value
		}
	}
	point.values = evaluation.internValues(next)
	return point
}

func (calls *CallSites) projectedCaptureValues(summary ExitSummary, projection captureProjection) []exitValue {
	values := decodeTuple(calls.captureModeTuple(summary, projection.invocation, projection.mode))
	if len(values) != summary.captureArity || projection.outcomes&UnknownExit != 0 {
		return make([]exitValue, summary.captureArity)
	}
	return values
}
