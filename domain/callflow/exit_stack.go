package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// An unwind stack is a finite continuation over the four nonterminal modes.
// Composition preserves LIFO order without bounding loop registrations.
type exitStack [4]Outcomes

func (stack exitStack) apply(outcomes Outcomes) Outcomes {
	result := outcomes & (Aborts | Diverges | UnknownExit)
	for i, mode := range pendingExitModes() {
		if outcomes&mode != 0 {
			result |= stack[i]
		}
	}
	return result
}

func (stack exitStack) push(calls *CallSites, summaries ExitSummaries, invocationRef pdg.Ref) exitStack {
	return stack.pushValues(calls, summaries, invocationRef, calls.defaultValues())
}

func (stack exitStack) pushValues(calls *CallSites, summaries ExitSummaries, invocationRef pdg.Ref, context bindingflow.Evaluation) exitStack {
	var next exitStack
	for i, mode := range pendingExitModes() {
		outcomes := calls.outcomesWithValues(summaries, invocationRef, mode == Panics || mode == PanicDuringGoexit, context)
		next[i] = stack.apply(invocationOutcomes(mode, outcomes))
	}
	return next
}

func pendingExitModes() [4]Outcomes {
	return [4]Outcomes{Returns, Panics, PanicDuringGoexit, ExitsGoroutine}
}

func identityExitStack() exitStack { return exitStack(pendingExitModes()) }

func invocationOutcomes(pending, completed Outcomes) Outcomes {
	result := completed & (Aborts | Diverges | UnknownExit)
	for _, outcome := range []Outcomes{Returns, RecoversInherited, Panics, PanicDuringGoexit, ExitsGoroutine} {
		if completed&outcome == 0 {
			continue
		}
		next, err := deferflow.ResolveInvocation(exitResult(pending), exitResult(outcome), outcome == RecoversInherited)
		if err != nil {
			result |= UnknownExit
			continue
		}
		result |= resultOutcome(next)
	}
	return result
}

func exitResult(outcome Outcomes) deferflow.ExitResult[struct{}] {
	switch outcome {
	case Returns, RecoversInherited:
		return deferflow.ExitResult[struct{}]{Kind: deferflow.ReturnExit, Complete: true}
	case Panics, PanicDuringGoexit:
		return deferflow.ExitResult[struct{}]{Kind: deferflow.PanicExit, ResumeGoexit: outcome == PanicDuringGoexit, Complete: true}
	case ExitsGoroutine:
		return deferflow.ExitResult[struct{}]{Kind: deferflow.GoexitExit, ResumeGoexit: true, Complete: true}
	default:
		return deferflow.ExitResult[struct{}]{}
	}
}

func resultOutcome(result deferflow.ExitResult[struct{}]) Outcomes {
	switch result.Kind {
	case deferflow.ReturnExit:
		return Returns
	case deferflow.PanicExit:
		if result.ResumeGoexit {
			return PanicDuringGoexit
		}
		return Panics
	case deferflow.GoexitExit:
		return ExitsGoroutine
	default:
		return UnknownExit
	}
}
