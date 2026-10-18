package resourceflow

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// CleanupRule describes an acquisition's existing advisory release protocol.
type CleanupRule struct {
	Method, Field string
	Receiver      bool
}

// RecognizesDeferredCleanup checks saved release inputs on all modeled exits.
// Unknown ordinary calls remain unproved. This is presence evidence for the
// missing-defer detector, never proof that foreign code returns or preserves state.
func (q *Query) RecognizesDeferredCleanup(call pdg.Ref, rule CleanupRule, exits callflow.ExitSummaries) bool {
	if !q.Reachable(call) {
		return false
	}
	policy := Policy{recognizeDeferred: true, releaseMethod: rule.Method, Exits: exits}
	if rule.Receiver {
		policy.Resource = q.symbol(q.paths.Operand(call, 1))
		return q.After(call, policy)
	}
	owner := q.acquisitionOwner(call)
	if owner == 0 {
		return false
	}
	policy = q.cleanupPolicy(owner, policy)
	if rule.Field != "" && !q.cleanupField(&policy, rule.Field) {
		return false
	}
	return q.After(owner, policy)
}

func (q *Query) acquisitionOwner(call pdg.Ref) pdg.Ref {
	for _, edge := range q.graph.Edges {
		if edge.Source != call || edge.Kind != pdg.Data || edge.Position != 1 {
			continue
		}
		if q.acquisitionBindingOwner(call, edge.Target) {
			return edge.Target
		}
	}
	return 0
}

func (q *Query) cleanupField(policy *Policy, field string) bool {
	for i, node := range q.graph.Nodes {
		if node.MatchLabel == 0 {
			continue
		}
		label := q.graph.Tables.MatchLabels[node.MatchLabel-1]
		if q.graph.Tables.Text(label.Detail) != field {
			continue
		}
		path := q.paths.Path(pdg.Ref(i+1), q.facts)
		if path.Root == policy.Resource && len(path.Fields) == 1 {
			policy.resourceFields = path.Fields
			return true
		}
	}
	return false
}

func (q *Query) acquisitionBindingOwner(call, owner pdg.Ref) bool {
	for _, write := range q.paths.Writes(owner) {
		if write.Position == 1 && q.singleBindingInput(owner, call) && q.bindsResource(owner, write.Symbol) {
			return true
		}
	}
	return false
}

func (q *Query) cleanupPolicy(owner pdg.Ref, policy Policy) Policy {
	writes := q.paths.Writes(owner)
	for _, write := range writes {
		switch write.Position {
		case 1:
			policy.Resource = write.Symbol
		case 2:
			// The existing acquisition rules with two results return (resource,error).
			// Pipe has three results; its second handle is not an acquisition error.
			if len(writes) == 2 {
				policy.AcquisitionError = write.Symbol
			}
		}
	}
	return policy
}

func (q *Query) canReturn(start int) bool {
	queue := []int{start}
	seen := make([]bool, len(q.flow.Blocks))
	for i := 0; i < len(queue); i++ {
		block := queue[i]
		if seen[block] {
			continue
		}
		seen[block] = true
		if q.returnBlock(block) {
			return true
		}
		for _, next := range q.flow.Blocks[block].Successors {
			queue = append(queue, int(next-1))
		}
	}
	return false
}

func (q *Query) returnBlock(block int) bool {
	for _, ref := range q.flow.Blocks[block].Operations {
		if q.graph.Nodes[ref-1].Category == pdg.Return {
			return true
		}
	}
	return false
}

// Presence evidence can follow an already saved cleanup across a loop with a
// return path. It does not prove termination. A closed loop has no such exit.
func (p *proof) recognizesLoopCleanup(site point, machine deferflow.Machine[bool]) bool {
	if !p.policy.recognizeDeferred || !p.query.canReturn(site.block) {
		return false
	}
	return p.unwind(machine, false)
}

// A saved value receiver survives later binding writes. Keep walking to check
// later defers and aborts, but stop using the overwritten acquisition guard.
func (p *proof) invalidated(ref pdg.Ref, machine deferflow.Machine[bool]) bool {
	if !p.query.invalidates(ref, p.policy) {
		return false
	}
	if !p.policy.recognizeDeferred || !p.unwind(machine, false) {
		return true
	}
	p.policy.AcquisitionError = 0
	p.policy.aliases = map[pdg.Ref]bool{}
	return false
}
