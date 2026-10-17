package callflow

import "github.com/rstyled/gstyle/domain/pdg"

type exitPass uint8

func (pass exitPass) update(old, outcomes Outcomes) Outcomes {
	if pass == finitePass {
		return (old | outcomes) &^ Diverges
	}
	return old&^Diverges | outcomes&Diverges
}

const (
	finitePass exitPass = iota
	infinitePass
)

// Finite completions grow from bottom. Possible divergence then descends from
// top with those completions fixed. Registration alone is not execution.
func solveExitBodies(bodies map[string]*exitBody) ExitSummaries {
	summaries := seedExitSummaries(bodies)
	seedCallInputs(bodies, summaries)
	for {
		setInputPass(summaries, finitePass)
		solveExitPass(bodies, summaries, finitePass)
		seedPossibleDivergence(bodies, summaries)
		count := invocationCount(summaries)
		solveExitPass(bodies, summaries, infinitePass)
		if invocationCount(summaries) == count {
			break
		}
	}
	sealInputs(summaries)
	return summaries
}

func solveExitPass(bodies map[string]*exitBody, summaries ExitSummaries, pass exitPass) {
	for changed := true; changed; {
		changed = false
		count := invocationCount(summaries)
		for name, body := range bodies {
			old := summaries[name]
			next := ExitSummary{
				Ordinary:      pass.update(old.Ordinary, evaluateBody(body, summaries, false)),
				DeferredPanic: pass.update(old.DeferredPanic, evaluateBody(body, summaries, true)),
				Receiver:      old.Receiver,
				invocations:   old.invocations,
			}
			summaries[name] = next
			inputChanged := solveInputPass(body, summaries, old.invocations, pass)
			changed = changed || old != next || inputChanged
		}
		changed = changed || invocationCount(summaries) != count
	}
}

func seedExitSummaries(bodies map[string]*exitBody) ExitSummaries {
	summaries := ExitSummaries{}
	for name, body := range bodies {
		seed := UnknownExit
		if body != nil && body.flow != nil {
			seed = 0
		}
		summaries[name] = ExitSummary{Ordinary: seed, DeferredPanic: seed, Receiver: bodyHasReceiver(body), invocations: newExitInvocations(body)}
	}
	return summaries
}

func seedPossibleDivergence(bodies map[string]*exitBody, summaries ExitSummaries) {
	for name, body := range bodies {
		if body == nil || body.flow == nil {
			continue
		}
		old := summaries[name]
		seedInputDivergence(old.invocations)
		summaries[name] = ExitSummary{Ordinary: old.Ordinary | Diverges, DeferredPanic: old.DeferredPanic | Diverges, Receiver: old.Receiver, invocations: old.invocations}
	}
}

func bodyHasReceiver(body *exitBody) bool {
	if body == nil {
		return false
	}
	for _, input := range body.graph.Function.Inputs {
		if input.Role == pdg.ReceiverRole {
			return true
		}
	}
	return false
}
