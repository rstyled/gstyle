package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestDeferredContinuationComposition(t *testing.T) {
	for _, tc := range []struct{ pending, child, want Outcomes }{
		{Returns, Returns, Returns}, {Panics, Returns, Panics}, {Panics, RecoversInherited, Returns},
		{PanicDuringGoexit, RecoversInherited, ExitsGoroutine}, {ExitsGoroutine, Panics, PanicDuringGoexit},
		{Returns, PanicDuringGoexit, PanicDuringGoexit}, {Panics, ExitsGoroutine, ExitsGoroutine},
		{Panics, Aborts, Aborts}, {Panics, Diverges, Diverges}, {Panics, UnknownExit, UnknownExit},
		{Panics, Returns | RecoversInherited, Panics | Returns},
	} {
		if got := invocationOutcomes(tc.pending, tc.child); got != tc.want {
			t.Fatalf("%+v: got %v", tc, got)
		}
	}
}
func TestUnwindContinuationOrder(t *testing.T) {
	calls := &CallSites{targets: map[pdg.Ref]callTargets{1: {names: []string{"recoverer"}, known: true}, 2: {names: []string{"panic"}, known: true}}}
	summaries := ExitSummaries{"recoverer": {Ordinary: Returns, DeferredPanic: RecoversInherited}}
	evaluation := newExitEvaluation(&exitBody{calls: calls}, summaries)
	base := singleCompletion(completion{mode: Returns})
	olderRecovery := evaluation.applyDeferred(evaluation.applyDeferred(base, 2, bindingflow.Evaluation{}), 1, bindingflow.Evaluation{})
	olderPanic := evaluation.applyDeferred(evaluation.applyDeferred(base, 1, bindingflow.Evaluation{}), 2, bindingflow.Evaluation{})
	if got := olderRecovery.outcomes(); got != Returns {
		t.Fatalf("older recoverer must consume newer panic: %v", got)
	}
	if got := olderPanic.outcomes(); got != Panics {
		t.Fatalf("earlier recovery cannot consume later panic: %v", got)
	}
}
