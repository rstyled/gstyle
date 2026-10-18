package resourceflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

// Handoffs are protocol boundaries, not a proof that the recipient closes the
// handle. They must actually occur before any path is considered discharged.
func (q *Query) callHandoff(ref pdg.Ref, policy Policy) bool {
	if !q.calls.EntersFunction(ref) {
		return false
	}
	if _, known := q.paths.InputValues(ref); !known {
		return false
	}
	positions, known := q.calls.ResourceInputs(ref)
	if !known {
		return false
	}
	for _, position := range positions {
		if q.resource(q.paths.Operand(ref, position), policy) {
			return true
		}
	}
	return false
}

func (q *Query) deferredHandoff(invocation deferflow.Invocation[bool]) bool {
	if !q.calls.EntersFunction(invocation.Node) {
		return false
	}
	positions, known := q.calls.ResourceInputs(invocation.Node)
	if !known {
		return false
	}
	for _, capture := range invocation.Captures {
		if capturedHandoff(capture, positions) {
			return true
		}
	}
	return false
}

func (q *Query) globalHandoff(ref pdg.Ref, policy Policy) bool {
	if !bindingOperation(q.graph.Nodes[ref-1].Category) {
		return false
	}
	for _, write := range q.paths.Writes(ref) {
		if q.globalResourceWrite(write, policy) {
			return true
		}
	}

	return false
}

func (q *Query) globalBinding(symbol pdg.Ref) bool {
	return symbol != 0 && q.graph.Symbols[symbol-1].Role == pdg.GlobalRole
}

func (q *Query) globalResourceWrite(write bindingflow.Write, policy Policy) bool {
	return q.globalBinding(write.Symbol) && write.Location == 0 && q.resource(write.Source, policy)
}

// Explicit value constructors can retain a handle. Arbitrary computations and
// call results are not identity/carrier proofs merely because they depend on it.
func (q *Query) carriedResource(ref pdg.Ref, policy Policy, seen map[pdg.Ref]bool) bool {
	if q.resource(ref, policy) {
		return true
	}
	if ref == 0 || seen[ref] {
		return false
	}
	seen[ref] = true
	if !q.carrierOperation(ref) {
		return false
	}
	inputs, known := q.paths.InputValues(ref)
	if !known {
		return false
	}
	return q.inputsCarryResource(inputs, policy, seen)
}

func (q *Query) inputsCarryResource(inputs []pdg.Ref, policy Policy, seen map[pdg.Ref]bool) bool {
	for _, input := range inputs {
		if q.carriedResource(input, policy, seen) {
			return true
		}
	}
	return false
}

func (q *Query) carrierOperation(ref pdg.Ref) bool {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	switch q.graph.Tables.Text(attrs.Operator.Value) {
	case "composite", "key-value":
		return true
	default:
		return false
	}
}

func (p *proof) actionStep(ref pdg.Ref, machine deferflow.Machine[bool]) int {
	if p.directRelease(ref) {
		return 1
	}
	if !p.policy.TransferCalls {
		return 0
	}
	if p.query.callHandoff(ref, p.policy) {
		return 1
	}
	if !p.query.globalHandoff(ref, p.policy) {
		return 0
	}
	if p.outcomeStep(p.operationOutcomes(ref), machine) < 0 {
		return -1
	}
	return 1
}

func (p *proof) directRelease(ref pdg.Ref) bool {
	return !p.policy.recognizeDeferred && p.query.releaseCall(ref, p.policy)
}

func capturedHandoff(capture deferflow.SavedCapture[bool], positions []pdg.Position) bool {
	return capture.Kind == "argument" && capture.Value && slices.Contains(positions, capture.Position)
}
