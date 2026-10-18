package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Queries prepares one graph's immutable indexes. Consumers share these queries
// and metadata; their value bindings, defer frames and proof states stay local.
type Queries struct {
	Bindings   *bindingflow.Query
	Calls      *CallSites
	Operations *OperationEffects
	Facts      bindingflow.Facts
	Deferred   deferflow.Index
}

func NewQueries(graph *pdg.Graph) (Queries, error) {
	calls, err := NewCallSites(graph)
	if err != nil {
		return Queries{}, err
	}
	if calls == nil {
		return Queries{}, nil
	}
	operations, err := newOperationEffects(graph, calls.paths)
	if err != nil {
		return Queries{}, err
	}
	return Queries{Bindings: calls.paths, Calls: calls, Operations: operations, Facts: calls.facts, Deferred: calls.deferred}, nil
}
