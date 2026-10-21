package resourceflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Bound path/state expansion; an incomplete search returns no observations.
const maxCleanupObservationStates = 4096

type cleanupObservationState struct {
	block   pdg.Ref
	aliases map[pdg.Ref]bool
}
type cleanupObservationKey struct {
	block   pdg.Ref
	aliases string
}

// ObservedDeferredReleases follows saved receiver/argument values through local
// binding copies. It does not prove registration on every path or eventual
// cleanup execution. Unsupported bindings and exhausted searches add no evidence.
func (q *Query) ObservedDeferredReleases(acquisition pdg.Ref) []pdg.Ref {
	if !q.Observed(acquisition) || q.graph.ValueFlow == nil {
		return nil
	}
	sites := q.prepared.Deferred.Sites()
	if len(sites) == 0 {
		return nil
	}
	matches := q.observeCleanupBindings(acquisition)
	var calls []pdg.Ref
	for _, site := range sites {
		if matches[site.Registration] {
			calls = append(calls, site.Invocation)
		}
	}
	return calls
}

func (q *Query) observeCleanupBindings(acquisition pdg.Ref) map[pdg.Ref]bool {
	flow := q.graph.ValueFlow
	pending := []cleanupObservationState{{block: flow.Entry, aliases: map[pdg.Ref]bool{}}}
	seen := map[cleanupObservationKey]bool{}
	matches := map[pdg.Ref]bool{}
	for len(pending) > 0 {
		if len(seen) >= maxCleanupObservationStates {
			return nil
		}
		state := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		key := cleanupObservationKey{state.block, aliasKey(state.aliases)}
		if seen[key] {
			continue
		}
		seen[key] = true
		aliases := q.observeCleanupBlock(flow.Blocks[state.block-1], state.aliases, acquisition, matches)
		for _, next := range flow.Blocks[state.block-1].Successors {
			pending = append(pending, cleanupObservationState{next, aliases})
		}
	}
	return matches
}

func (q *Query) observeCleanupBlock(block pdg.FlowBlock, aliases map[pdg.Ref]bool, acquisition pdg.Ref, matches map[pdg.Ref]bool) map[pdg.Ref]bool {
	for _, ref := range block.Operations {
		policy := Policy{aliases: aliases}
		if site, found := q.machine.Site(ref); found {
			match := q.observedSavedCleanup(site, policy)
			previous, visited := matches[site.Registration]
			matches[site.Registration] = match && (!visited || previous)
		}
		aliases = q.observedAliases(ref, acquisition, policy)
	}
	return aliases
}

func (q *Query) observedAliases(ref, acquisition pdg.Ref, policy Policy) map[pdg.Ref]bool {
	writes := q.paths.Writes(ref)
	if len(writes) == 0 {
		return policy.aliases
	}
	if unknownBindingWrite(writes) {
		return map[pdg.Ref]bool{}
	}
	aliases := q.nextAliases(writes, policy)
	for _, write := range writes {
		if !q.observationBinding(write.Symbol) {
			delete(aliases, write.Symbol)
			continue
		}
		if observedAcquisitionWrite(write, acquisition) {
			aliases[write.Symbol] = true
		}
	}
	return aliases
}

// Captured or addressed cells can change outside the visible local writes.
func (q *Query) observationBinding(symbol pdg.Ref) bool {
	if !q.facts.SingleCellSymbols[symbol] || q.facts.CapturedSymbols[symbol] {
		return false
	}
	policy := Policy{Resource: symbol}
	for i, node := range q.graph.Nodes {
		if q.addressesTracked(pdg.Ref(i+1), q.graph.NodeAttributes(node), policy) {
			return false
		}
	}
	return true
}

func (q *Query) observedSavedCleanup(site deferflow.Site, policy Policy) bool {
	captures := make([]deferflow.SavedCapture[bool], len(site.Captures))
	for i, capture := range site.Captures {
		captures[i] = deferflow.SavedCapture[bool]{Capture: capture, Value: q.resource(capture.Source, policy)}
	}
	return q.deferredRelease(deferflow.Invocation[bool]{Node: site.Invocation, Captures: captures}, Policy{})
}

func observedAcquisitionWrite(write bindingflow.Write, acquisition pdg.Ref) bool {
	return write.Source == acquisition && write.Position == 1 && write.Location == 0 && !write.Unresolved
}
