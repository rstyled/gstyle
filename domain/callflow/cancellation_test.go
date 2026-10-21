package callflow

import (
	"context"
	"testing"
)

type cancelingExitReader struct{ cancel context.CancelFunc }

func (reader cancelingExitReader) lookupSummary(string) (ExitSummary, bool) {
	reader.cancel()
	return ExitSummary{Ordinary: Returns, DeferredPanic: Returns}, true
}

func TestCancellationDuringExitEvaluation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := choiceExitBodies(1, 1)["f0"]
	body.ctx = ctx
	evaluation := newExitEvaluation(body, cancelingExitReader{cancel})
	if got := evaluation.run(false, ""); got != UnknownExit || ctx.Err() == nil {
		t.Fatalf("got %v, context error %v", got, ctx.Err())
	}
	cache := bodyEvaluationCache{}
	cache.evaluate(bodyEvaluationKey{body: body}, ExitSummaries{})
	if len(cache) != 0 {
		t.Fatal("canceled evaluation was cached")
	}
}
