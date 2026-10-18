package callflow

import (
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
	olderRecovery := identityExitStack().push(calls, summaries, 1).push(calls, summaries, 2)
	olderPanic := identityExitStack().push(calls, summaries, 2).push(calls, summaries, 1)
	if got := olderRecovery.apply(Returns); got != Returns {
		t.Fatalf("older recoverer must consume newer panic: %v", got)
	}
	if got := olderPanic.apply(Returns); got != Panics {
		t.Fatalf("earlier recovery cannot consume later panic: %v", got)
	}
}
