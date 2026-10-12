package pdg

import (
	"strings"
	"testing"
)

func flowFixture() *Graph {
	pool := NewBuilder()
	attrs := []Attributes{{Reads: []Access{{Symbol: 1}}}}
	graph := &Graph{Tables: pool.Tables, Nodes: []Node{{Category: Read, Attributes: 1}}, Symbols: []Symbol{{Role: ParameterRole}}, Function: Function{ReferenceCount: CountFact{Status: Known, Value: 1}}, NormalFlow: &NormalFlow{Entry: 1, ReadBindings: 1, Blocks: []FlowBlock{{Operations: []Ref{1}, Reachable: true}}}}
	if err := PackAttributes(graph, attrs); err != nil {
		panic(err)
	}
	return graph
}

func TestNormalFlowContextRejectsInvalidClaims(t *testing.T) {
	cases := []struct {
		name, want string
		change     func(*Graph)
	}{
		{"entry", "entry", func(g *Graph) { g.NormalFlow.Entry = 2 }},
		{"successor", "successor", func(g *Graph) { g.NormalFlow.Blocks[0].Successors = []Ref{2} }},
		{"operation", "operation", func(g *Graph) { g.NormalFlow.Blocks[0].Operations = []Ref{2} }},
		{"reachability", "reachability", func(g *Graph) { g.NormalFlow.Blocks[0].Reachable = false }},
		{"duplicate", "multiple CFG sites", func(g *Graph) { g.NormalFlow.Blocks[0].Operations = []Ref{1, 1} }},
		{"value-entity", "value entity", func(g *Graph) { g.Nodes[0].Category = Merge }},
		{"cycle", "cyclic", func(g *Graph) { g.NormalFlow.Blocks[0].Successors = []Ref{1} }},
		{"context-count", "reference count", func(g *Graph) { g.NormalFlow.ReadBindings = 0 }},
		{"producer-count", "PDG reference count", func(g *Graph) { g.Function.ReferenceCount.Value = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := flowFixture()
			tc.change(g)
			err := validateNormalFlow(g)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v; want %s", err, tc.want)
			}
		})
	}
	if err := validateNormalFlow(flowFixture()); err != nil {
		t.Fatal(err)
	}
}
