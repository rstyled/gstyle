package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (calls *CallSites) returnedValues(summaries ExitSummaries, ref pdg.Ref, context bindingflow.Evaluation) string {
	targets := calls.targets[ref]
	if !targets.known {
		return ""
	}
	var returned string
	unknown := false
	for _, name := range targets.names {
		values, missing := calls.targetReturned(name, summaries, exitInvocationContext{ref: ref, values: context})
		returned = joinReturned(returned, values)
		unknown = unknown || missing
	}
	if unknown {
		return unknownReturned(len(returned))
	}
	return returned
}

func (calls *CallSites) targetReturned(name string, summaries ExitSummaries, invocation exitInvocationContext) (string, bool) {
	summary, found := summaries[name]
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
	if len(values) != summary.arity {
		return unknownReturned(summary.arity), false
	}
	return values, false
}

func (calls *CallSites) contextReturned(summary ExitSummary, invocation exitInvocationContext) string {
	key := calls.inputKey(summary.invocations, invocation.ref, invocation.values)
	if key == "" {
		return summary.returned
	}
	state, found := summary.invocations.lookup(key)
	if !found {
		return summary.returned
	}
	return state.returned
}
