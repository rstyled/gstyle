package callflow

func bodyCaptureCount(body *exitBody) int {
	if body == nil {
		return 0
	}
	return len(body.captures)
}

func (evaluation *exitEvaluation) captureTuple(point exitPoint) string {
	values := make([]byte, len(evaluation.body.captures))
	bindings := evaluation.values[point.values].bindings
	for i, symbol := range evaluation.body.captures {
		values[i] = byte(bindings[symbol])
	}
	return string(values)
}

func (calls *CallSites) contextCaptureEffects(summary ExitSummary, invocation exitInvocationContext) captureEffects {
	index := 0
	if invocation.inherited {
		index = 1
	}
	key := calls.inputKey(summary.invocations, invocation.ref, invocation.values)
	if key == "" {
		return summary.captured[index]
	}
	state, found := summary.invocations.lookup(key)
	if !found {
		return summary.captured[index]
	}
	return state.captured[index]
}
