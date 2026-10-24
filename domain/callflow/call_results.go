package callflow

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
