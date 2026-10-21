package resourceflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Observe prepares typed canonical inputs when sequential flow is unavailable.
// It supplies observations only: no release/transfer proof can use this query.
func Observe(graph *pdg.Graph) (*Query, error) {
	if graph == nil {
		return nil, fmt.Errorf("resource observation requires a graph")
	}
	prepared, err := callflow.NewQueries(graph)
	if err != nil {
		return nil, err
	}
	query := &Query{prepared: prepared, graph: graph, paths: prepared.Bindings, facts: prepared.Facts, machine: deferflow.NewMachineWithIndex[bool](prepared.Deferred), methods: map[pdg.Ref]string{}}
	for index, node := range graph.Nodes {
		if err := query.indexFacts(pdg.Ref(index+1), graph.NodeAttributes(node)); err != nil {
			return nil, err
		}
	}
	return query, nil
}

func (q *Query) HasFlow() bool { return q != nil && q.flow != nil }

// Observed is deliberately weaker than Reachable on a partial graph.
func (q *Query) Observed(ref pdg.Ref) bool {
	if q.HasFlow() {
		return q.Reachable(ref)
	}
	return q != nil && ref != 0 && int(ref) <= len(q.graph.Nodes)
}
