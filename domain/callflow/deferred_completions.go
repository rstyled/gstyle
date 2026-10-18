package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (evaluation *exitEvaluation) applyDeferred(suffix completions, ref pdg.Ref, saved bindingflow.Evaluation) completions {
	result := completions{}
	for value := range suffix.all() {
		if value.mode&(Aborts|Diverges|UnknownExit) != 0 {
			result.insert(value)
			continue
		}
		result.add(evaluation.deferredCompletion(value, ref, saved))
	}
	return result
}

func (evaluation *exitEvaluation) deferredCompletion(value completion, ref pdg.Ref, saved bindingflow.Evaluation) completions {
	point := evaluation.completionPoint(value)
	// Only lexical cells are live. Explicit arguments keep their registration
	// bindings, operand snapshots and tuple results.
	saved.Captures = evaluation.callCellValues(evaluation.values[value.values])
	saved.ClosureCells = evaluation.concreteHeap(evaluation.values[value.values])
	saved.RepeatedCells = evaluation.values[value.values].repeatedHeap()
	invocation := exitInvocationContext{ref: ref, inherited: value.mode == Panics || value.mode == PanicDuringGoexit, values: saved}
	evaluation.observeInvocation(invocation)
	outcomes := evaluation.body.calls.outcomesWithValues(evaluation.summaries, ref, invocation.inherited, saved)
	result := terminalCompletion(outcomes & (Aborts | Diverges | UnknownExit))
	result.removeZero()
	for _, mode := range completedModes() {
		if outcomes&mode == 0 {
			continue
		}
		next := evaluation.transferInvocation(point, invocation, mode)
		result.add(evaluation.resolvedDeferred(value, next, mode))
	}
	return result
}

func (evaluation *exitEvaluation) resolvedDeferred(value completion, point exitPoint, mode Outcomes) completions {
	outcomes := invocationOutcomes(value.mode, mode)
	result := completions{}
	for completed := range evaluation.finish(point, outcomes).all() {
		if completed.mode&(Aborts|Diverges|UnknownExit) == 0 {
			completed.returned = value.returned
		}
		result.insert(completed)
	}
	return result
}
