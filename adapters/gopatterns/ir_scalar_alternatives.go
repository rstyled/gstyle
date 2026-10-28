package gopatterns

import (
	"encoding/json"
	"go/token"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (b *irBuilder) shortCircuitAlternatives() {
	if !b.scalarDefinitions {
		return
	}
	joins := map[pdg.Ref]pdg.Ref{}
	for expression, predicate := range b.logicalPredicates {
		joins[b.nodes[expression]] = predicate
	}
	for i := range b.graph.Edges {
		edge := &b.graph.Edges[i]
		predicate := joins[edge.Target]
		if predicate != 0 && b.pool.Tables.Text(edge.Subkind) == "scalar-value" {
			*edge = b.shortCircuitAlternative(*edge, predicate)
		}
	}
}

func (b *irBuilder) shortCircuitAlternative(edge pdg.Edge, predicate pdg.Ref) pdg.Edge {
	attrs := b.attributes[b.graph.Nodes[predicate-1].Attributes-1]
	selected := (edge.Position == 2) == (b.pool.Tables.Text(attrs.Operator.Value) == token.LAND.String())
	outcome, position := "false", pdg.Position(2)
	if selected {
		outcome, position = "true", 1
	}
	guard := map[string]string{"controller_pdg_node_id": b.pool.Tables.Text(b.graph.Nodes[predicate-1].ID), "outcome": outcome}
	data, _ := json.Marshal([][]map[string]string{{guard}})
	b.graph.Annotations = append(b.graph.Annotations, b.pool.SingletonExtension("gstyle.go:guards", string(data)))
	return pdg.Edge{
		ID: edge.ID, Source: edge.Source, Target: edge.Target, Kind: edge.Kind,
		Evidence: edge.Evidence, Provenance: edge.Provenance,
		Symbol: edge.Symbol, Definition: edge.Definition, Location: edge.Location,
		LoopCarried: edge.LoopCarried,
		Subkind:     b.pool.Text("value_alternative"), Outcome: b.pool.Text(outcome),
		Position: position, Annotations: pdg.Ref(len(b.graph.Annotations)),
	}
}
