package resourceflow

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *proof) callStep(ref pdg.Ref, machine deferflow.Machine[bool]) int {
	outcomes := p.query.calls.Outcomes(p.policy.Exits, ref, false)
	return p.outcomeStep(outcomes, machine)
}

func (p *proof) outcomeStep(outcomes callflow.Outcomes, machine deferflow.Machine[bool]) int {
	if !usableOutcomes(outcomes) {
		return -1
	}
	if !p.exceptionalCalls(machine, outcomes) {
		return -1
	}
	if outcomes&callflow.Returns != 0 {
		return 0
	}
	return 1
}

func (p *proof) exceptionalCalls(machine deferflow.Machine[bool], outcomes callflow.Outcomes) bool {
	for _, outcome := range []callflow.Outcomes{callflow.Panics, callflow.PanicDuringGoexit, callflow.ExitsGoroutine} {
		if outcomes&outcome == 0 {
			continue
		}
		branch := machine
		if branch.BeginPropagation(completedExit(outcome)) != nil {
			return false
		}
		if !p.drain(branch, false) {
			return false
		}
	}
	return outcomes&callflow.RecoversInherited == 0
}

func (p *proof) deferredCall(machine deferflow.Machine[bool], invocation deferflow.Invocation[bool], transfer bool) bool {
	inherited := machine.Exit().Kind == deferflow.PanicExit
	outcomes := p.query.calls.Outcomes(p.policy.Exits, invocation.Node, inherited)
	if !usableOutcomes(outcomes) {
		return false
	}
	// Exit facts do not establish preservation of named results or alias state.
	transfer = transfer && p.query.deferredReturns(invocation.Node)
	return p.deferredOutcomes(machine, outcomes, transfer)
}

func (p *proof) deferredOutcomes(machine deferflow.Machine[bool], outcomes callflow.Outcomes, transfer bool) bool {
	for _, outcome := range []callflow.Outcomes{callflow.Returns, callflow.RecoversInherited, callflow.Panics, callflow.PanicDuringGoexit, callflow.ExitsGoroutine} {
		if outcomes&outcome == 0 {
			continue
		}
		branch := machine
		if completeInvocation(&branch, outcome) != nil {
			return false
		}
		if !p.drain(branch, transfer) {
			return false
		}
	}
	return true
}

func usableOutcomes(outcomes callflow.Outcomes) bool {
	allowed := callflow.Returns | callflow.RecoversInherited | callflow.Panics | callflow.PanicDuringGoexit | callflow.ExitsGoroutine
	return outcomes != 0 && outcomes&^allowed == 0
}

func completeInvocation(machine *deferflow.Machine[bool], outcome callflow.Outcomes) error {
	if outcome == callflow.RecoversInherited {
		_, _, err := machine.CompleteRecovered()
		return err
	}
	return machine.CompletePropagation(completedExit(outcome))
}

func completedExit(outcome callflow.Outcomes) deferflow.ExitResult[bool] {
	switch outcome {
	case callflow.Returns:
		return deferflow.ExitResult[bool]{Kind: deferflow.ReturnExit, Complete: true}
	case callflow.Panics, callflow.PanicDuringGoexit:
		return deferflow.ExitResult[bool]{Kind: deferflow.PanicExit, ResumeGoexit: outcome == callflow.PanicDuringGoexit, Complete: true}
	case callflow.ExitsGoroutine:
		return deferflow.ExitResult[bool]{Kind: deferflow.GoexitExit, ResumeGoexit: true, Complete: true}
	default:
		return deferflow.ExitResult[bool]{}
	}
}
