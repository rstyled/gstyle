package callflow

import "github.com/rstyled/gstyle/domain/pdg"

// Invocation proofs descend from top as contexts and reachable prefixes grow.
// They remain separate from outcome/result evidence, which grows from bottom.
type memoryProof struct{ preserves, confined bool }

func (proof memoryProof) meet(next memoryProof) memoryProof {
	return memoryProof{preserves: proof.preserves && next.preserves, confined: proof.confined && next.confined}
}
func (calls *CallSites) invocationMemory(summaries ExitSummaries, invocation exitInvocationContext) memoryProof {
	targets := calls.invocationTargets(invocation.ref, invocation.values)
	if !targets.known || !calls.resolvedSiteWrites(invocation.ref) {
		return memoryProof{}
	}
	if targets.nilValue {
		return memoryProof{preserves: true, confined: true}
	}
	if len(targets.names) == 0 {
		return memoryProof{}
	}
	proof := memoryProof{preserves: true, confined: true}
	for _, name := range targets.names {
		proof = proof.meet(calls.targetInvocationMemory(name, summaries, invocation))
	}
	return proof
}
func (calls *CallSites) targetInvocationMemory(name string, summaries ExitSummaries, invocation exitInvocationContext) memoryProof {
	if targetPreservesMemory(name, summaries) {
		return memoryProof{preserves: true, confined: true}
	}
	summary, found := summaries[name]
	if !found {
		return memoryProof{}
	}
	proof := calls.contextMemory(summary, invocation)
	if proof.preserves {
		return proof
	}
	if summary.captureArity != 0 && !calls.linkedFrame(summary) {
		proof.confined = false
	}
	return proof
}
func (calls *CallSites) contextMemory(summary ExitSummary, invocation exitInvocationContext) memoryProof {
	key := calls.inputKey(summary.invocations, invocation.ref, invocation.values)
	if key != "" {
		if state, found := summary.invocations.lookup(key); found {
			return state.memory
		}
	}
	return memoryProof{preserves: summary.PreservesCallerMemory, confined: summary.frameConfined}
}
func (calls *CallSites) linkedFrame(summary ExitSummary) bool {
	if summary.invocations == nil {
		return false
	}
	return summary.invocations.captureBindings(calls.graph) != nil
}
func (evaluation *exitEvaluation) observeInvocation(invocation exitInvocationContext) {
	evaluation.memory = evaluation.memory.meet(evaluation.body.calls.invocationMemory(evaluation.summaries, invocation))
}
func (evaluation *exitEvaluation) observeMemory(ref pdg.Ref) {
	body := evaluation.body
	if body.graph.Nodes[ref-1].Category == pdg.Call {
		return
	}
	if body.operations.effect(ref).Unknown {
		evaluation.memory = memoryProof{}
		return
	}
	evaluation.memory = evaluation.memory.meet(memoryProof{preserves: body.privateWrites(ref), confined: body.frameWrites(ref)})
}
