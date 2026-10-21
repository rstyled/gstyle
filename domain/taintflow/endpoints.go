package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

type endpoints uint8

const (
	sourceEndpoint endpoints = 1 << iota
	sinkEndpoint
)

// A finding requires a modeled source and sink somewhere in the catalog.
// They can belong to different functions or be connected through callbacks.
func hasEndpoints(queries map[string]*query) bool {
	var found endpoints
	for _, query := range queries {
		for index, node := range query.graph.Nodes {
			if node.Category != pdg.Call {
				continue
			}
			found |= query.endpoints(pdg.Ref(index + 1))
			if found == sourceEndpoint|sinkEndpoint {
				return true
			}
		}
	}
	return false
}

func (q *query) endpoints(ref pdg.Ref) endpoints {
	var found endpoints
	if q.models.Source != nil && q.models.Source(q.graph, ref) {
		found |= sourceEndpoint
	}
	if q.models.Sinks != nil && len(q.models.Sinks(q.graph, ref)) != 0 {
		found |= sinkEndpoint
	}
	return found
}
