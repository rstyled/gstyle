package callflow

import (
	"context"
	"github.com/rstyled/gstyle/domain/pdg"
)

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
	return solveExitBodiesContext(context.Background(), bodies)
}
func solveExitBodiesContext(ctx context.Context, bodies map[string]*exitBody) ExitSummaries {
	summaries := seedExitSummaries(bodies)
	seedCallInputs(bodies, summaries)
	solver := exitPassSolver{ctx: ctx, summaries: summaries, cache: bodyEvaluationCache{}}
	for ctx.Err() == nil {
		setInputPass(summaries, finitePass)
		solver.pass = finitePass
		solver.solve(bodies)
		seedPossibleDivergence(bodies, summaries)
		count := invocationCount(summaries)
		solver.pass = infinitePass
		solver.solve(bodies)
		if invocationCount(summaries) == count {
			break
		}
	}
	sealInputs(summaries)
	return summaries
}

type exitPassSolver struct {
	ctx       context.Context
	summaries ExitSummaries
	pass      exitPass
	cache     bodyEvaluationCache
}

func (solver exitPassSolver) solve(bodies map[string]*exitBody) {
	for changed := true; changed; {
		changed = false
		count := invocationCount(solver.summaries)
		for name, body := range bodies {
			if solver.ctx.Err() != nil {
				return
			}
			changed = solver.solveBody(name, body) || changed
		}
		changed = changed || invocationCount(solver.summaries) != count
	}
}

func (solver exitPassSolver) solveBody(name string, body *exitBody) bool {
	old := solver.summaries[name]
	ordinary := solver.cache.evaluate(bodyEvaluationKey{body: body}, solver.summaries)
	inherited := solver.cache.evaluate(bodyEvaluationKey{body: body, inherited: true}, solver.summaries)
	next := ExitSummary{
		heaps:                 [2]heapEffects{solver.pass.heapEffects(old.heaps[0], ordinary.heaps), solver.pass.heapEffects(old.heaps[1], inherited.heaps)},
		frameConfined:         old.frameConfined,
		captureArity:          old.captureArity,
		captured:              [2]captureEffects{solver.pass.captureEffects(old.captured[0], ordinary.captured), solver.pass.captureEffects(old.captured[1], inherited.captured)},
		PreservesCallerMemory: old.PreservesCallerMemory,
		returned:              solver.pass.returned(old.returned, ordinary.returned), arity: old.arity,
		Ordinary:      solver.pass.update(old.Ordinary, ordinary.outcomes),
		DeferredPanic: solver.pass.update(old.DeferredPanic, inherited.outcomes),
		Receiver:      old.Receiver,
		invocations:   old.invocations,
	}
	solver.summaries[name] = next
	inputChanged := solver.solveInputs(body, old.invocations)
	return old != next || inputChanged
}

func seedExitSummaries(bodies map[string]*exitBody) ExitSummaries {
	summaries := ExitSummaries{}
	for name, body := range bodies {
		seed := UnknownExit
		if body != nil && body.flow != nil {
			seed = 0
		}
		summaries[name] = ExitSummary{captureArity: bodyCaptureCount(body), frameConfined: memoryBodyKnown(body), PreservesCallerMemory: memoryBodyKnown(body), Ordinary: seed, DeferredPanic: seed, Receiver: bodyHasReceiver(body), arity: bodyResultCount(body), invocations: newExitInvocations(body)}
	}
	solveMemoryPreservation(bodies, summaries)
	solveFrameConfinement(bodies, summaries)
	return summaries
}

func seedPossibleDivergence(bodies map[string]*exitBody, summaries ExitSummaries) {
	for name, body := range bodies {
		if body == nil || body.flow == nil {
			continue
		}
		old := summaries[name]
		seedInputDivergence(old.invocations)
		old.Ordinary |= Diverges
		old.DeferredPanic |= Diverges
		summaries[name] = old
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
