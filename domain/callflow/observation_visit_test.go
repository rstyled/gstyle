package callflow

import (
	"context"
	"github.com/rstyled/gstyle/domain/pdg"
	"reflect"
	"testing"
)

// Compare the forward visitor with the existing completion traversal on every
// directed call graph of up to three functions, including cycles and joins.
func TestObservationTraversalMatchesCompletions(t *testing.T) {
	for size := 1; size <= 3; size++ {
		for mask := 0; mask < 1<<(size*size); mask++ {
			bodies := choiceExitBodies(size, mask)
			summaries := solveExitBodies(bodies)
			for name, body := range bodies {
				want := collectTestObservations(body, summaries, false)
				got := collectTestObservations(body, summaries, true)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("size=%d mask=%d function=%s: got %+v, want %+v", size, mask, name, got, want)
				}
			}
		}
	}
}

func collectTestObservations(body *exitBody, summaries ExitSummaries, observing bool) Observations {
	evaluation := newExitEvaluation(body, summaries)
	evaluation.observing = observing
	evaluation.callOutcomes = map[pdg.Ref]Outcomes{}
	evaluation.operationOutcomes = map[pdg.Ref]Outcomes{}
	evaluation.conditions = map[pdg.Ref]uint8{}
	evaluation.run(false, "")
	return Observations{Calls: evaluation.callOutcomes, Operations: evaluation.operationOutcomes, Conditions: conditionFacts(evaluation.conditions)}
}

func TestObservationCancellationDuringCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := choiceExitBodies(1, 1)["f0"]
	body.ctx = ctx
	evaluation := newExitEvaluation(body, cancelingExitReader{cancel})
	evaluation.observing = true
	evaluation.callOutcomes = map[pdg.Ref]Outcomes{}
	if got := evaluation.run(false, ""); got != UnknownExit || ctx.Err() == nil {
		t.Fatalf("got %v, context error %v", got, ctx.Err())
	}
}
