package callflow

// Summary reads include context-sensitive invocation reads and phase flags.
// The cache belongs to one analysis of immutable body graphs.
type exitSummaryReader interface {
	lookupSummary(string) (ExitSummary, bool)
}

func (summaries ExitSummaries) lookupSummary(name string) (ExitSummary, bool) {
	summary, found := summaries[name]
	return summary, found
}

func readExitSummary(reader exitSummaryReader, name string) (ExitSummary, bool) {
	if reader == nil {
		return ExitSummary{}, false
	}
	return reader.lookupSummary(name)
}

type invocationObservation struct {
	value exitInvocation
	found bool
}

type summaryObservation struct {
	original ExitSummary
	view     ExitSummary
	found    bool
	inputs   map[string]invocationObservation
	pass     exitPass
	building bool
}

type observedSummaries struct {
	source ExitSummaries
	reads  map[string]*summaryObservation
}

func (observed observedSummaries) lookupSummary(name string) (ExitSummary, bool) {
	if read, found := observed.reads[name]; found {
		return read.view, read.found
	}
	summary, found := observed.source[name]
	read := observeSummary(summary, found)
	observed.reads[name] = read
	return read.view, found
}

func observeSummary(summary ExitSummary, found bool) *summaryObservation {
	if summary.invocations == nil {
		return &summaryObservation{original: summary, view: summary, found: found}
	}
	inputs := map[string]invocationObservation{}
	// Share the solver-owned stores; keep read observations in this scoped view.
	invocations := *summary.invocations
	invocations.observed = inputs
	view := summary
	view.invocations = &invocations
	return &summaryObservation{
		original: summary, view: view, found: found, inputs: inputs,
		pass: invocations.pass, building: invocations.building,
	}
}

func (read *summaryObservation) unchanged(summary ExitSummary, found bool) bool {
	if read.found != found || read.original != summary {
		return false
	}
	return read.samePhase(summary.invocations) && read.sameInputs(summary.invocations)
}

func (read *summaryObservation) sameInputs(invocations *exitInvocations) bool {
	for key, old := range read.inputs {
		value, present := invocations.states[key]
		if old.found != present || old.value != value {
			return false
		}
	}
	return true
}

func (read *summaryObservation) samePhase(invocations *exitInvocations) bool {
	if invocations == nil {
		return true
	}
	return read.pass == invocations.pass && read.building == invocations.building
}

type bodyEvaluationKey struct {
	body      *exitBody
	inputs    string
	inherited bool
}

type cachedBodyEvaluation struct {
	result evaluatedBody
	reads  map[string]*summaryObservation
}

func (entry cachedBodyEvaluation) current(summaries ExitSummaries) bool {
	for name, read := range entry.reads {
		summary, found := summaries[name]
		if !read.unchanged(summary, found) {
			return false
		}
	}
	return true
}

type bodyEvaluationCache map[bodyEvaluationKey]cachedBodyEvaluation

func (cache bodyEvaluationCache) evaluate(key bodyEvaluationKey, summaries ExitSummaries) evaluatedBody {
	if key.inherited && !key.body.recoversInherited() {
		key.inherited = false
	}
	if entry, found := cache[key]; found && entry.current(summaries) {
		return entry.result
	}
	observed := observedSummaries{source: summaries, reads: map[string]*summaryObservation{}}
	result := evaluateBodyResult(key.body, observed, key.inherited, key.inputs)
	cache[key] = cachedBodyEvaluation{result: result, reads: observed.reads}
	return result
}

// Only direct recover operations inspect the frame's inherited panic flag.
// Other calls receive the pending completion of this frame's own execution.
func (body *exitBody) recoversInherited() bool {
	if body == nil || body.calls == nil {
		return false
	}
	for ref := range body.calls.targets {
		if body.calls.recoveryCall(ref) {
			return true
		}
	}
	return false
}
