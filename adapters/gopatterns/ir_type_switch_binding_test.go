package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestTypeSwitchClausesHaveDistinctReachingBindings(t *testing.T) {
	graph := parseIR(t, `package p;func f(x any)any{switch v:=x.(type){case int:return v;case string:return v;default:return v}}`, pdg.NewBuilder())
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.Locations) != 0 {
		t.Fatal("type-switch guard invents an unknown memory write")
	}
	bindings := map[pdg.Ref]pdg.Ref{}
	for i, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if graph.Tables.Text(attrs.Construct) != "type-switch-binding" {
			continue
		}
		bindings[attrs.Writes[0].Symbol] = pdg.Ref(i + 1)
	}
	if len(bindings) != 3 {
		t.Fatalf("got %d clause bindings, want 3", len(bindings))
	}
	reads := 0
	for _, edge := range graph.Edges {
		if graph.Tables.Text(edge.Subkind) != "reaching-definition" {
			continue
		}
		binding := bindings[edge.Symbol]
		if binding == 0 {
			continue
		}
		if edge.Source != binding {
			t.Fatal("case value comes from a different clause")
		}
		reads++
	}
	if reads != 3 {
		t.Fatalf("got %d case reads with definitions, want 3", reads)
	}
	for _, edge := range graph.Edges {
		if graph.Tables.Text(edge.Subkind) == "type-switch-value" {
			delete(bindings, graph.NodeAttributes(graph.Nodes[edge.Target-1]).Writes[0].Symbol)
		}
	}
	if len(bindings) != 0 {
		t.Fatal("clause definitions lack switched-value dependencies")
	}
}
