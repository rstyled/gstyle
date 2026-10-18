package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

func (calls *CallSites) capturedWrites(summaries ExitSummaries, ref pdg.Ref, context bindingflow.Evaluation) (map[pdg.Ref]exitValue, bool) {
	if !calls.confinedTargets(summaries, ref) {
		return nil, false
	}
	var alternatives []map[pdg.Ref]exitValue
	for _, name := range calls.targets[ref].names {
		invocation := exitInvocationContext{ref: ref, values: context}
		outcomes := calls.targetWithValues(name, summaries, invocation)
		if outcomes&(Returns|UnknownExit) != 0 {
			alternatives = append(alternatives, calls.targetCaptureWrites(name, summaries, invocation, outcomes))
		}
	}
	return joinCaptureWrites(alternatives, context.Bindings), true
}

func (calls *CallSites) targetCaptureWrites(name string, summaries ExitSummaries, invocation exitInvocationContext, outcomes Outcomes) map[pdg.Ref]exitValue {
	summary := summaries[name]
	result := map[pdg.Ref]exitValue{}
	if targetPreservesMemory(name, summaries) || summary.captureArity == 0 {
		return result
	}
	values := calls.contextCaptured(summary, invocation)
	if len(values) != summary.captureArity || outcomes&UnknownExit != 0 {
		values = unknownReturned(summary.captureArity)
	}
	links := summary.invocations.captureBindings(calls.graph)
	for i, symbol := range summary.invocations.captures {
		result[links[symbol]] = exitValue(values[i])
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
	next := exitValues{bindings: maps.Clone(old.bindings), cells: maps.Clone(old.cells), nodes: old.nodes, tuples: old.tuples, unorderedReads: old.unorderedReads}
	if next.cells == nil {
		next.cells = map[pdg.Ref]exitValue{}
	}
	if next.bindings == nil {
		next.bindings = map[pdg.Ref]exitValue{}
	}
	for symbol, value := range writes {
		next.cells[symbol] = joinCellValue(next.cells, symbol, value)
		delete(next.bindings, symbol)
		if evaluation.body.facts.SingleCellSymbols[symbol] {
			next.bindings[symbol] = value
		}
	}
	point.values = evaluation.internValues(next)
	return point
}
