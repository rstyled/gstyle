package resourceflow

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type acquisitionProof struct {
	query     *Query
	call      pdg.Ref
	start     point
	policy    Policy
	relevant  map[int]bool
	active    map[point]deferflow.StateKey
	completed map[stateKey]bool
}

func newAcquisitionProof(query *Query, call pdg.Ref, start point, policy Policy) *acquisitionProof {
	return &acquisitionProof{query: query, call: call, start: start, policy: policy, relevant: query.blocksReaching(call), active: map[point]deferflow.StateKey{}, completed: map[stateKey]bool{}}
}

func (analysis *acquisitionProof) walk(site point, machine deferflow.Machine[bool]) bool {
	if !analysis.relevant[site.block] {
		return true
	}
	key := stateKey{point: site, stack: machine.StateKey()}
	if active, found := analysis.active[site]; found {
		return active == key.stack
	}
	if result, known := analysis.completed[key]; known {
		return result
	}
	analysis.active[site] = key.stack
	result := analysis.step(site, machine)
	delete(analysis.active, site)
	analysis.completed[key] = result
	return result
}

func (analysis *acquisitionProof) step(site point, machine deferflow.Machine[bool]) bool {
	block := analysis.query.flow.Blocks[site.block]
	if site.offset == len(block.Operations) {
		return analysis.successors(block, machine)
	}
	ref := block.Operations[site.offset]
	if ref == analysis.call {
		return analysis.query.proveWithMachine(analysis.start, analysis.policy, machine)
	}
	switch analysis.prefixOperation(ref, &machine) {
	case -1:
		return false
	case 1:
		return true // No acquisition return occurs on this path.
	}
	site.offset++
	return analysis.walk(site, machine)
}

func (analysis *acquisitionProof) prefixOperation(ref pdg.Ref, machine *deferflow.Machine[bool]) int {
	if site, found := machine.Site(ref); found {
		if analysis.query.deferredReturns(site.Invocation) {
			return 0
		}
		if machine.Register(ref, make([]bool, len(site.Captures))) != nil {
			return -1
		}
		return 0
	}
	if analysis.query.graph.Nodes[ref-1].Category == pdg.Return {
		return 1
	}
	if analysis.query.graph.Nodes[ref-1].Category != pdg.Call {
		return 0
	}
	return prefixCallStep(analysis.query.calls.Outcomes(analysis.policy.Exits, ref, false))
}

func (analysis *acquisitionProof) successors(block pdg.FlowBlock, machine deferflow.Machine[bool]) bool {
	for _, next := range block.Successors {
		if !analysis.walk(point{block: int(next - 1), present: true}, machine) {
			return false
		}
	}
	return true
}

// AfterAcquisition starts ownership at a call's normal return. Operand effects
// and the result binding must not be skipped, nor may earlier defers disappear.
func (q *Query) AfterAcquisition(call, owner pdg.Ref, policy Policy) bool {
	if !q.validAcquisition(call, owner, policy.Resource) {
		return false
	}
	if !q.acquisitionBindingCompletes(call, policy) {
		return false
	}
	site := q.sites[owner-1]
	start := point{block: site.block, offset: site.offset + 1, present: site.present}
	releaseOnly := Policy{Resource: policy.Resource, AcquisitionError: policy.AcquisitionError, TransferCalls: policy.TransferCalls, Helpers: policy.Helpers, Exits: policy.Exits}
	// Cleanup proved with no older frames runs before any older frame can act.
	if q.prove(start, releaseOnly) {
		return true
	}
	analysis := newAcquisitionProof(q, call, start, policy)
	return analysis.walk(point{block: int(q.flow.Entry - 1), present: true}, q.machine)
}

func (q *Query) validAcquisition(call, owner, resource pdg.Ref) bool {
	if !q.Reachable(call) || !q.Reachable(owner) {
		return false
	}
	if q.graph.Nodes[call-1].Category != pdg.Call {
		return false
	}
	return q.acquisitionBinding(call, owner, resource)
}

func (q *Query) acquisitionBinding(call, owner, resource pdg.Ref) bool {
	if !bindingOperation(q.graph.Nodes[owner-1].Category) {
		return false
	}
	if !q.singleBindingInput(owner, call) {
		return false
	}
	return q.bindsResource(owner, resource) && q.operations.SameRegion(call, owner)
}

func (q *Query) bindsResource(owner, resource pdg.Ref) bool {
	if resource == 0 {
		return false
	}
	found := false
	for _, write := range q.graph.NodeAttributes(q.graph.Nodes[owner-1]).Writes {
		if write.Position != 1 {
			continue
		}
		if found || !directResourceWrite(write, resource) {
			return false
		}
		found = true
	}
	return found
}

func (q *Query) acquisitionBindingCompletes(call pdg.Ref, policy Policy) bool {
	candidates, known := q.operations.LaterOperations(call)
	if !known {
		return false
	}
	for _, ref := range candidates {
		if q.acquisitionOperandOutcomes(ref, policy) != callflow.Returns {
			return false
		}
	}
	return true
}

func (q *Query) acquisitionOperandOutcomes(ref pdg.Ref, policy Policy) callflow.Outcomes {
	if q.graph.Nodes[ref-1].Category == pdg.Call {
		return q.calls.Outcomes(policy.Exits, ref, false)
	}
	// The fresh result is not yet bound; old variable values cannot prove it safe.
	return q.operations.Outcomes(ref)
}

func (q *Query) blocksReaching(call pdg.Ref) map[int]bool {
	predecessors := make([][]int, len(q.flow.Blocks))
	for from, block := range q.flow.Blocks {
		for _, next := range block.Successors {
			predecessors[next-1] = append(predecessors[next-1], from)
		}
	}
	seen := map[int]bool{}
	collectReachingBlocks(q.sites[call-1].block, predecessors, seen)
	return seen
}

func (q *Query) singleBindingInput(owner, call pdg.Ref) bool {
	values, known := q.paths.InputValues(owner)
	return known && len(values) == 1 && values[0] == call
}

func bindingOperation(category pdg.Category) bool {
	return category == pdg.Assignment || category == pdg.Declaration
}

func directResourceWrite(write pdg.Access, resource pdg.Ref) bool {
	return write.Symbol == resource && write.Location == 0 && write.UnresolvedReason == 0
}

func prefixCallStep(outcomes callflow.Outcomes) int {
	if outcomes == 0 || outcomes&(callflow.Returns|callflow.UnknownExit) != 0 {
		return 0
	}
	return 1
}

func collectReachingBlocks(block int, predecessors [][]int, seen map[int]bool) {
	if seen[block] {
		return
	}
	seen[block] = true
	for _, previous := range predecessors[block] {
		collectReachingBlocks(previous, predecessors, seen)
	}
}
