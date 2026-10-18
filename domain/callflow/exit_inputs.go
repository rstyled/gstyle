package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type exitInvocationContext struct {
	ref       pdg.Ref
	inherited bool
	values    bindingflow.Evaluation
}

type exitInvocation struct{ Ordinary, DeferredPanic Outcomes }

type exitInvocations struct {
	inputs   []pdg.Parameter
	states   map[string]exitInvocation
	pass     exitPass
	building bool
}

func newExitInvocations(body *exitBody) *exitInvocations {
	if body == nil || body.flow == nil {
		return nil
	}
	inputs := body.graph.Function.Inputs
	_, _, known := bindingflow.CallSignature(inputs)
	if len(inputs) == 0 || !known {
		return nil
	}
	return &exitInvocations{inputs: inputs, states: map[string]exitInvocation{}, building: true}
}

func (invocations *exitInvocations) lookup(key string) (exitInvocation, bool) {
	if state, found := invocations.states[key]; found {
		return state, true
	}
	if !invocations.building {
		return exitInvocation{}, false
	}
	var seed Outcomes
	if invocations.pass == infinitePass {
		seed = Diverges
	}
	state := exitInvocation{Ordinary: seed, DeferredPanic: seed}
	invocations.states[key] = state
	return state, true
}

func (calls *CallSites) defaultValues() bindingflow.Evaluation {
	if calls == nil {
		return bindingflow.Evaluation{}
	}
	return bindingflow.Evaluation{Facts: calls.facts}
}

func (calls *CallSites) targetWithValues(name string, summaries ExitSummaries, invocation exitInvocationContext) Outcomes {
	summary := summaries[name]
	key := calls.inputKey(summary.invocations, invocation.ref, invocation.values)
	if key == "" {
		return targetOutcomes(name, summaries, invocation.inherited)
	}
	state, known := summary.invocations.lookup(key)
	if !known {
		return summary.outcomes(invocation.inherited)
	}
	return (ExitSummary{Ordinary: state.Ordinary, DeferredPanic: state.DeferredPanic, Receiver: summary.Receiver}).outcomes(invocation.inherited)
}

func (calls *CallSites) inputKey(invocations *exitInvocations, ref pdg.Ref, context bindingflow.Evaluation) string {
	if invocations == nil || calls.paths == nil {
		return ""
	}
	values, known := calls.paths.CallValues(ref, invocations.inputs, context)
	if !known {
		return ""
	}
	return calls.knownInputKey(invocations.inputs, values)
}

func (calls *CallSites) knownInputKey(inputs []pdg.Parameter, arguments []bindingflow.Value) string {
	values := make([]byte, len(inputs))
	known := false
	for i, input := range inputs {
		if input.Symbol == 0 {
			continue
		}
		value := abstractValue(arguments[i])
		values[i] = byte(value)
		known = known || value != unknownValue
	}
	if !known {
		return ""
	}
	return string(values)
}

func inputBindings(body *exitBody, key string) map[pdg.Ref]exitValue {
	bindings := map[pdg.Ref]exitValue{}
	if len(key) != len(body.graph.Function.Inputs) {
		return bindings
	}
	for i, input := range body.graph.Function.Inputs {
		if key[i] != 0 && input.Symbol != 0 {
			bindings[input.Symbol] = exitValue(key[i])
		}
	}
	return bindings
}

func invocationCount(summaries ExitSummaries) int {
	count := 0
	for _, summary := range summaries {
		if summary.invocations != nil {
			count += len(summary.invocations.states)
		}
	}
	return count
}

func solveInputPass(body *exitBody, summaries ExitSummaries, invocations *exitInvocations, pass exitPass) bool {
	if invocations == nil {
		return false
	}
	changed := false
	for key, old := range invocations.states {
		next := exitInvocation{
			Ordinary:      pass.update(old.Ordinary, evaluateBodyInputs(body, summaries, false, key)),
			DeferredPanic: pass.update(old.DeferredPanic, evaluateBodyInputs(body, summaries, true, key)),
		}
		if old == next {
			continue
		}
		invocations.states[key] = next
		changed = true
	}
	return changed
}

func seedInputDivergence(invocations *exitInvocations) {
	if invocations == nil {
		return
	}
	invocations.pass = infinitePass
	for key, state := range invocations.states {
		invocations.states[key] = exitInvocation{Ordinary: state.Ordinary | Diverges, DeferredPanic: state.DeferredPanic | Diverges}
	}
}

func sealInputs(summaries ExitSummaries) {
	for _, summary := range summaries {
		if summary.invocations != nil {
			summary.invocations.building = false
		}
	}
}

// Consumers can start after an acquisition or another opaque prefix. Prepare
// constant-input queries even when the caller's full exit flow cannot get there.
// Computing a callee context does not assert that its call site executes.
func seedCallInputs(bodies map[string]*exitBody, summaries ExitSummaries) {
	for _, body := range bodies {
		if body == nil || body.calls == nil {
			continue
		}
		for ref := range body.calls.targets {
			body.calls.Outcomes(summaries, ref, false)
		}
	}
}

func setInputPass(summaries ExitSummaries, pass exitPass) {
	for _, summary := range summaries {
		if summary.invocations != nil {
			summary.invocations.pass = pass
		}
	}
}
