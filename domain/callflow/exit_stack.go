package callflow

import "github.com/rstyled/gstyle/domain/deferflow"

func pendingExitModes() [4]Outcomes {
	return [4]Outcomes{Returns, Panics, PanicDuringGoexit, ExitsGoroutine}
}

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
