package callflow

import "github.com/rstyled/gstyle/domain/pdg"

func (calls *CallSites) captureModeTuple(summary ExitSummary, invocation exitInvocationContext, mode Outcomes) string {
	index := completionIndex(mode)
	if index < 0 {
		return unknownReturned(summary.captureArity)
	}
	return calls.contextCaptureEffects(summary, invocation)[index]
}

func (evaluation *exitEvaluation) transferInvocation(point exitPoint, invocation exitInvocationContext, mode Outcomes) exitPoint {
	projection := evaluation.body.calls.projectEnvironment(evaluation.summaries, invocation, mode)
	proof := evaluation.body.calls.invocationMemory(evaluation.summaries, invocation)
	return evaluation.applyEnvironmentProjection(point, invocationCompletion{invocation, projection, proof, true})
}

func (evaluation *exitEvaluation) callCompletions(point exitPoint, ref pdg.Ref, outcomes Outcomes) completions {
	result := evaluation.finish(point, outcomes&(Aborts|Diverges|UnknownExit))
	for _, mode := range []Outcomes{Panics, PanicDuringGoexit, ExitsGoroutine} {
		if outcomes&mode == 0 {
			continue
		}
		invocation := exitInvocationContext{ref: ref, values: evaluation.valueContext(point)}
		next := evaluation.transferInvocation(point, invocation, mode)
		result.add(evaluation.finish(next, mode))
	}
	if outcomes&Returns != 0 || (evaluation.callOutcomes != nil && outcomes&UnknownExit != 0) {
		result.add(evaluation.visit(evaluation.transferValues(point, ref)))
	}
	return result
}
