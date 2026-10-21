package resourceflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// InputAcquisition requires every canonical reaching value of this argument to
// be the same acquisition. Aliases with mixed or missing origins stay unknown.
func (q *Query) InputAcquisition(call pdg.Ref, position pdg.Position, acquisition pdg.Ref) bool {
	if !q.Observed(call) || !q.Observed(acquisition) {
		return false
	}
	return acquisitionOrigins(q.paths.OriginsAt(call, position, q.facts), acquisition)
}

func acquisitionOrigins(origins bindingflow.Origins, acquisition pdg.Ref) bool {
	if !origins.Known || len(origins.Sources) == 0 {
		return false
	}
	for _, source := range origins.Sources {
		if !acquisitionSource(source, acquisition) {
			return false
		}
	}
	return true
}

func acquisitionSource(source bindingflow.ValueSource, acquisition pdg.Ref) bool {
	return source.Node == acquisition && source.Result <= 1 && !source.Zero && source.Representation == bindingflow.UnchangedRepresentation
}

// InputCleanup recognizes a saved release on the selected parameter on modeled
// exits. It is advisory presence evidence and does not prove worker termination.
func (q *Query) InputCleanup(position pdg.Position, exits callflow.ExitSummaries) bool {
	if !q.HasFlow() || position == 0 || int(position) > len(q.graph.Function.Inputs) || len(q.protocol.Release) == 0 {
		return false
	}
	policy := Policy{recognizeDeferred: true, Resource: q.graph.Function.Inputs[position-1].Symbol, Exits: exits}
	return q.FromEntry(policy)
}
