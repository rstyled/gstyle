package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestSummaryObservationsInvalidateChangedEvidence(t *testing.T) {
	for _, field := range []string{"outcome", "return", "memory", "capture", "heap", "missing"} {
		t.Run(field, func(t *testing.T) {
			summaries := ExitSummaries{"callee": {Ordinary: Returns}}
			observed := observedSummaries{source: summaries, reads: map[string]*summaryObservation{}}
			readExitSummary(observed, "callee")
			entry := cachedBodyEvaluation{reads: observed.reads}
			if !entry.current(summaries) {
				t.Fatal("unchanged summary invalidates result")
			}
			next := summaries["callee"]
			switch field {
			case "outcome":
				next.Ordinary = Panics
			case "return":
				next.returned = encodeTuple([]exitValue{trueValue})
			case "memory":
				next.PreservesCallerMemory = true
			case "capture":
				next.captured[0][0] = "changed"
			case "heap":
				next.heaps[0][0] = "changed"
			case "missing":
				delete(summaries, "callee")
			}
			if field != "missing" {
				summaries["callee"] = next
			}
			if entry.current(summaries) {
				t.Fatal("changed evidence reuses stale result")
			}
		})
	}
}

func TestSummaryObservationsTrackMissingTargets(t *testing.T) {
	summaries := ExitSummaries{}
	observed := observedSummaries{source: summaries, reads: map[string]*summaryObservation{}}
	readExitSummary(observed, "late")
	entry := cachedBodyEvaluation{reads: observed.reads}
	summaries["unrelated"] = ExitSummary{Ordinary: Panics}
	if !entry.current(summaries) {
		t.Fatal("unread target invalidates result")
	}
	summaries["late"] = ExitSummary{Ordinary: Returns}
	if entry.current(summaries) {
		t.Fatal("newly resolved target reuses unknown result")
	}
}

func TestSummaryObservationsTrackOnlyReadContexts(t *testing.T) {
	inputs := &exitInvocations{states: map[string]exitInvocation{"used": {Ordinary: Returns}, "other": {Ordinary: Panics}}, building: true}
	summaries := ExitSummaries{"callee": {invocations: inputs}}
	observed := observedSummaries{source: summaries, reads: map[string]*summaryObservation{}}
	summary, _ := readExitSummary(observed, "callee")
	summary.invocations.lookup("used")
	entry := cachedBodyEvaluation{reads: observed.reads}
	inputs.states["other"] = exitInvocation{Ordinary: Returns}
	if !entry.current(summaries) {
		t.Fatal("unread context invalidates result")
	}
	inputs.states["used"] = exitInvocation{Ordinary: Panics}
	if entry.current(summaries) {
		t.Fatal("changed context reuses stale result")
	}
	// Preserve the first read when the same context changes during evaluation.
	summary.invocations.lookup("used")
	if entry.current(summaries) {
		t.Fatal("later read overwrites the dependency snapshot")
	}
}

func TestObservedContextRegistrationUsesSharedStore(t *testing.T) {
	inputs := &exitInvocations{states: map[string]exitInvocation{}, building: true, pass: infinitePass}
	summaries := ExitSummaries{"callee": {invocations: inputs}}
	observed := observedSummaries{source: summaries, reads: map[string]*summaryObservation{}}
	summary, _ := readExitSummary(observed, "callee")
	value, found := summary.invocations.lookup("new")
	if !found || value.Ordinary != Diverges {
		t.Fatal("context registration loses phase seed")
	}
	if inputs.states["new"] != value {
		t.Fatal("observation hides the registered context from the solver")
	}
	entry := cachedBodyEvaluation{reads: observed.reads}
	if !entry.current(summaries) {
		t.Fatal("registered context immediately invalidates its result")
	}
	inputs.states["new"] = exitInvocation{Ordinary: Returns}
	if entry.current(summaries) {
		t.Fatal("solved context does not invalidate its caller")
	}
}

func TestObservedMissingContextInvalidatesWhenAdded(t *testing.T) {
	inputs := &exitInvocations{states: map[string]exitInvocation{}}
	summaries := ExitSummaries{"callee": {invocations: inputs}}
	observed := observedSummaries{source: summaries, reads: map[string]*summaryObservation{}}
	summary, _ := readExitSummary(observed, "callee")
	if _, found := summary.invocations.lookup("missing"); found {
		t.Fatal("sealed store creates a context")
	}
	entry := cachedBodyEvaluation{reads: observed.reads}
	inputs.states["missing"] = exitInvocation{Ordinary: Returns}
	if entry.current(summaries) {
		t.Fatal("new context reuses a missing-context result")
	}
}

func TestBodyCacheInvalidatesTransitiveCalleeChange(t *testing.T) {
	bodies := choiceExitBodies(2, 1<<1) // f0 may call f1.
	summaries := seedExitSummaries(bodies)
	summaries["f1"] = ExitSummary{Ordinary: Panics, DeferredPanic: RecoversInherited}
	cache := bodyEvaluationCache{}
	key := bodyEvaluationKey{body: bodies["f0"]}
	first := cache.evaluate(key, summaries)
	if first.outcomes != Returns|Panics {
		t.Fatalf("first outcomes=%v", first.outcomes)
	}
	summaries["f1"] = ExitSummary{Ordinary: Aborts}
	second := cache.evaluate(key, summaries)
	if second.outcomes != Returns|Aborts {
		t.Fatalf("stale outcomes=%v", second.outcomes)
	}
	if len(cache) != 1 {
		t.Fatal("changed dependency creates an unrelated cache key")
	}
}

func TestSummaryObservationsTrackPhaseFlags(t *testing.T) {
	for _, phase := range []string{"pass", "building"} {
		t.Run(phase, func(t *testing.T) {
			inputs := &exitInvocations{states: map[string]exitInvocation{}, building: true}
			summaries := ExitSummaries{"callee": {invocations: inputs}}
			observed := observedSummaries{source: summaries, reads: map[string]*summaryObservation{}}
			readExitSummary(observed, "callee")
			entry := cachedBodyEvaluation{reads: observed.reads}
			if phase == "pass" {
				inputs.pass = infinitePass
			} else {
				inputs.building = false
			}
			if entry.current(summaries) {
				t.Fatal("phase change reuses a prior-phase evaluation")
			}
		})
	}
}

func TestBodyCacheSharesOnlyFramesWithoutRecover(t *testing.T) {
	body := choiceExitBodies(1, 0)["f0"]
	summaries := seedExitSummaries(map[string]*exitBody{"f0": body})
	cache := bodyEvaluationCache{}
	ordinary := cache.evaluate(bodyEvaluationKey{body: body}, summaries)
	inherited := cache.evaluate(bodyEvaluationKey{body: body, inherited: true}, summaries)
	if ordinary != inherited || len(cache) != 1 {
		t.Fatal("equivalent frames do not share evaluation")
	}
	recoverBody := &exitBody{calls: &CallSites{targets: map[pdg.Ref]callTargets{1: {known: true, names: []string{"recover"}}}}}
	cache.evaluate(bodyEvaluationKey{body: recoverBody}, summaries)
	cache.evaluate(bodyEvaluationKey{body: recoverBody, inherited: true}, summaries)
	if len(cache) != 3 {
		t.Fatal("recover frames lose distinct inherited state")
	}
}
