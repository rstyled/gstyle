package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (calls *CallSites) returnedValues(summaries exitSummaryReader, ref pdg.Ref, context bindingflow.Evaluation) string {
	targets := calls.invocationTargets(ref, context)
	if !targets.known {
		return ""
	}
	var returned string
	unknown := false
	for _, target := range targets.callables() {
		values, missing := calls.targetReturned(target.Name, summaries, calls.targetInvocation(summaries, exitInvocationContext{ref: ref, values: context}, target))
		returned = joinReturned(returned, values)
		unknown = unknown || missing
	}
	if unknown {
		return unknownReturned(len(decodeTuple(returned)))
	}
	return returned
}

func (calls *CallSites) targetReturned(name string, summaries exitSummaryReader, invocation exitInvocationContext) (string, bool) {
	summary, found := readExitSummary(summaries, name)
	outcomes := calls.targetWithValues(name, summaries, invocation)
	if !found || outcomes&UnknownExit != 0 {
		return "", true
	}
	if outcomes&Returns == 0 {
		return "", false
	}
	if summary.arity == 0 {
		return "", true
	}
	values := calls.contextReturned(summary, invocation)
	if len(decodeTuple(values)) != summary.arity {
		return unknownReturned(summary.arity), false
	}
	return values, false
}

func (calls *CallSites) contextReturned(summary ExitSummary, invocation exitInvocationContext) string {
	key := calls.inputKey(summary.invocations, invocation)
	if key == "" {
		return summary.returned
	}
	state, found := summary.invocations.lookup(key)
	if !found {
		return summary.returned
	}
	return state.returned
}
