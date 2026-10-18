package resourceflow

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type stateKey struct {
	errorSymbol pdg.Ref
	aliases     string
	point       point
	stack       deferflow.StateKey
}

type proof struct {
	query     *Query
	policy    Policy
	active    map[point]bool
	completed map[stateKey]bool
}

func (p *proof) walk(site point, machine deferflow.Machine[bool]) bool {
	if p.active[site] {
		return p.recognizesLoopCleanup(site, machine)
	}
	key := stateKey{point: site, stack: machine.StateKey(), aliases: aliasKey(p.policy.aliases), errorSymbol: p.policy.AcquisitionError}
	if result, known := p.completed[key]; known {
		return result
	}
	p.active[site] = true
	current := *p
	result := current.block(site, machine)
	delete(p.active, site)
	p.completed[key] = result
	return result
}

func (p *proof) block(site point, machine deferflow.Machine[bool]) bool {
	block := p.query.flow.Blocks[site.block]
	switch p.operations(block.Operations[site.offset:], &machine, &site.unstable) {
	case -1:
		return false
	case 1:
		return true
	}
	if len(block.Successors) == 0 {
		return false
	}
	return p.successors(block, machine, site.unstable)
}

func (p *proof) operations(refs []pdg.Ref, machine *deferflow.Machine[bool], unstable *bool) int {
	for _, ref := range refs {
		result := p.operation(ref, machine, unstable)
		if result != 0 {
			return result
		}
	}
	return 0
}

func (p *proof) successors(block pdg.FlowBlock, machine deferflow.Machine[bool], unstable bool) bool {
	for arm, next := range block.Successors {
		if p.failedAcquisition(block, arm) {
			continue
		}
		if !p.walk(point{block: int(next - 1), present: true, unstable: unstable}, machine) {
			return false
		}
	}
	return true
}

func (p *proof) operation(ref pdg.Ref, machine *deferflow.Machine[bool], unstable *bool) int {
	if !p.priorOperationEffects(ref, *machine) {
		return -1
	}
	if step := p.actionStep(ref, *machine); step != 0 {
		return step
	}
	if p.invalidated(ref, *machine) {
		return -1
	}
	if site, registration := machine.Site(ref); registration {
		if !p.save(site, machine) {
			return -1
		}
		return 0
	}
	return p.advanceOperation(ref, *machine, unstable)
}

func (p *proof) failedAcquisition(block pdg.FlowBlock, arm int) bool {
	if p.policy.AcquisitionError == 0 || len(block.Successors) != 2 {
		return false
	}
	subject, notNil := p.query.paths.NilPredicate(block.Condition, p.query.facts)
	if p.query.symbol(subject) != p.policy.AcquisitionError {
		return false
	}
	return (arm == 0) == notNil
}

func (p *proof) returnStep(ref pdg.Ref, machine deferflow.Machine[bool], stable bool) int {
	if p.unwind(machine, stable && p.query.returnedResource(ref, p.policy)) {
		return 1
	}
	return -1
}

func (p *proof) exitOperation(ref pdg.Ref, machine deferflow.Machine[bool], unstable *bool) int {
	if p.query.graph.Nodes[ref-1].Category == pdg.Return {
		return p.returnStep(ref, machine, !*unstable)
	}
	if p.query.graph.Nodes[ref-1].Category != pdg.Call {
		return p.outcomeStep(p.operationOutcomes(ref), machine)
	}
	result := p.callStep(ref, machine)
	if !p.query.deferredReturns(ref) {
		*unstable = true
	}
	return result
}

func (p *proof) operationOutcomes(ref pdg.Ref) callflow.Outcomes {
	input := p.query.paths.Operand(ref, 1)
	if p.query.resource(input, p.policy) || p.nonNilCleanupOwner(input) {
		return p.cleanupOutcomes(p.query.operations.OutcomesWithNonNilInput(ref))
	}
	return p.cleanupOutcomes(p.query.operations.Outcomes(ref))
}

func (p *proof) priorOperationEffects(ref pdg.Ref, machine deferflow.Machine[bool]) bool {
	if p.query.graph.Nodes[ref-1].Category != pdg.Call {
		return true
	}
	candidates, known := p.query.operations.PriorOperations(ref)
	if !known {
		// Missing evaluation-region evidence cannot establish that a saved defer is absent.
		return p.policy.recognizeDeferred
	}
	for _, candidate := range candidates {
		if p.outcomeStep(p.operationOutcomes(candidate), machine) < 0 {
			return false
		}
	}
	return true
}

// Store copies take effect only on the operation's normal continuation.
func (p *proof) advanceOperation(ref pdg.Ref, machine deferflow.Machine[bool], unstable *bool) int {
	result := p.exitOperation(ref, machine, unstable)
	if result == 0 {
		p.assignAliases(ref)
	}
	return result
}

func (p *proof) nonNilCleanupOwner(ref pdg.Ref) bool {
	return p.policy.recognizeDeferred && len(p.policy.resourceFields) != 0 && ownedSymbol(p.query.symbol(ref), p.policy)
}

func (q *Query) After(owner pdg.Ref, policy Policy) bool {
	if !q.Reachable(owner) {
		return false
	}
	site := q.sites[owner-1]
	site.offset++
	return q.prove(site, policy)
}

func (q *Query) FromEntry(policy Policy) bool {
	if q == nil {
		return false
	}
	return q.prove(point{block: int(q.flow.Entry - 1), present: true}, policy)
}

func (q *Query) prove(start point, policy Policy) bool {
	return q.proveWithMachine(start, policy, q.machine)
}

func (q *Query) proveWithMachine(start point, policy Policy, machine deferflow.Machine[bool]) bool {
	if policy.Resource == 0 {
		return false
	}
	if policy.aliases == nil {
		policy.aliases = map[pdg.Ref]bool{policy.Resource: true}
	}
	proof := proof{query: q, policy: policy, active: map[point]bool{}, completed: map[stateKey]bool{}}
	return proof.walk(start, machine)
}

func (q *Query) returnedResource(ref pdg.Ref, policy Policy) bool {
	if !policy.TransferReturn {
		return false
	}
	for i := range q.graph.Function.Outputs {
		position := pdg.Position(i + 1)
		input := q.paths.Operand(ref, position)
		if q.carriedResource(input, policy, map[pdg.Ref]bool{}) || (input == 0 && ownedSymbol(q.paths.ReadSymbol(ref, position), policy)) {
			return true
		}
	}
	return false
}
