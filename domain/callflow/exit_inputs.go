package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type exitInvocationContext struct {
	key       string
	keyReady  bool
	target    bindingflow.Callable
	ref       pdg.Ref
	inherited bool
	values    bindingflow.Evaluation
}

type exitInvocation struct {
	heaps                   [2]heapEffects
	memory                  memoryProof
	Ordinary, DeferredPanic Outcomes
	returned                string
	captured                [2]captureEffects
}

type exitInvocations struct {
	observed     map[string]invocationObservation
	graph        *pdg.Graph
	captures     []pdg.Ref
	environments map[*pdg.Graph]map[pdg.Ref]pdg.Ref
	inputs       []pdg.Parameter
	states       map[string]exitInvocation
	pass         exitPass
	building     bool
}

func newExitInvocations(body *exitBody) *exitInvocations {
	if body == nil || body.flow == nil {
		return nil
	}
	inputs := body.graph.Function.Inputs
	_, _, known := bindingflow.CallSignature(inputs)
	if len(inputs)+len(body.captures) == 0 || !known {
		return nil
	}
	return &exitInvocations{graph: body.graph, captures: body.captures, environments: map[*pdg.Graph]map[pdg.Ref]pdg.Ref{}, inputs: inputs, states: map[string]exitInvocation{}, building: true}
}

func (invocations *exitInvocations) lookup(key string) (exitInvocation, bool) {
	if state, found := invocations.states[key]; found {
		return invocations.observe(key, state, true)
	}
	if !invocations.building {
		return invocations.observe(key, exitInvocation{}, false)
	}
	var seed Outcomes
	if invocations.pass == infinitePass {
		seed = Diverges
	}
	state := exitInvocation{memory: memoryProof{preserves: true, confined: true}, Ordinary: seed, DeferredPanic: seed}
	invocations.states[key] = state
	return invocations.observe(key, state, true)
}

func (calls *CallSites) defaultValues() bindingflow.Evaluation {
	if calls == nil {
		return bindingflow.Evaluation{}
	}
	return bindingflow.Evaluation{Facts: calls.facts}
}

func (calls *CallSites) targetWithValues(name string, summaries exitSummaryReader, invocation exitInvocationContext) Outcomes {
	summary, _ := readExitSummary(summaries, name)
	if !calls.validTargetEnvironment(summary, invocation) {
		return UnknownExit
	}
	key := calls.inputKey(summary.invocations, invocation)
	if key == "" {
		return targetOutcomes(name, summaries, invocation.inherited)
	}
	state, known := summary.invocations.lookup(key)
	if !known {
		return summary.outcomes(invocation.inherited)
	}
	return (ExitSummary{Ordinary: state.Ordinary, DeferredPanic: state.DeferredPanic, Receiver: summary.Receiver}).outcomes(invocation.inherited)
}

func (calls *CallSites) inputKey(invocations *exitInvocations, invocation exitInvocationContext) string {
	if invocation.keyReady {
		return invocation.key
	}
	if invocations == nil || calls.paths == nil {
		return ""
	}
	return calls.environmentKey(invocations, invocation)
}

func (calls *CallSites) knownInputKey(inputs []pdg.Parameter, arguments []bindingflow.Value) string {
	values := make([]exitValue, len(inputs))
	known := false
	for i, input := range inputs {
		if input.Symbol == 0 {
			continue
		}
		value := abstractValue(arguments[i])
		values[i] = value
		known = known || value != unknownValue
	}
	if !known {
		return ""
	}
	return encodeTuple(values)
}

func inputBindings(body *exitBody, key string) map[pdg.Ref]exitValue {
	bindings := map[pdg.Ref]exitValue{}
	values := readEnvironmentInput(key).Values
	symbols := invocationSymbols(body)
	if len(values) != len(symbols) {
		return bindings
	}
	for i, symbol := range symbols {
		if values[i] != unknownValue && symbol != 0 {
			bindings[symbol] = values[i]
		}
	}
	return bindings
}

func invocationSymbols(body *exitBody) []pdg.Ref {
	symbols := make([]pdg.Ref, 0, len(body.graph.Function.Inputs)+len(body.captures))
	for _, input := range body.graph.Function.Inputs {
		symbols = append(symbols, input.Symbol)
	}
	return append(symbols, body.captures...)
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

func (solver exitPassSolver) solveInputs(body *exitBody, invocations *exitInvocations) bool {
	if invocations == nil {
		return false
	}
	changed := false
	for key, old := range invocations.states {
		if body.canceled() {
			return false
		}
		ordinary := solver.cache.evaluate(bodyEvaluationKey{body: body, inputs: key}, solver.summaries)
		inherited := solver.cache.evaluate(bodyEvaluationKey{body: body, inputs: key, inherited: true}, solver.summaries)
		next := exitInvocation{
			heaps:         [2]heapEffects{solver.pass.heapEffects(old.heaps[0], ordinary.heaps), solver.pass.heapEffects(old.heaps[1], inherited.heaps)},
			memory:        old.memory.meet(ordinary.memory).meet(inherited.memory),
			captured:      [2]captureEffects{solver.pass.captureEffects(old.captured[0], ordinary.captured), solver.pass.captureEffects(old.captured[1], inherited.captured)},
			returned:      solver.pass.returned(old.returned, ordinary.returned),
			Ordinary:      solver.pass.update(old.Ordinary, ordinary.outcomes),
			DeferredPanic: solver.pass.update(old.DeferredPanic, inherited.outcomes),
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
		state.Ordinary |= Diverges
		state.DeferredPanic |= Diverges
		invocations.states[key] = state
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

func (invocations *exitInvocations) observe(key string, state exitInvocation, found bool) (exitInvocation, bool) {
	if invocations.observed != nil {
		if _, recorded := invocations.observed[key]; !recorded {
			invocations.observed[key] = invocationObservation{value: state, found: found}
		}
	}
	return state, found
}
