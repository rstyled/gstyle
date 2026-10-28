package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestNormalFlowCountsReachableReadBindings(t *testing.T) {
	cases := []struct {
		body string
		want uint64
	}{
		{`return x+x`, 1},
		{`x=1; return 0`, 0},
		{`if c { x:=2; return x }; return x`, 3},
		{`return x; y=1; return y`, 1},
		{`if c { x=1 }; return x`, 2},
	}
	for _, tc := range cases {
		graph := parseIR(t, "package example; func f(x,y int,c bool) int { "+tc.body+" }", pdg.NewBuilder())
		if graph.NormalFlow == nil {
			t.Fatal("missing normal flow validation context")
		}
		if graph.Function.ReferenceCount.Status != pdg.Known || graph.Function.ReferenceCount.Value != tc.want {
			t.Fatalf("PDG count disagrees: %+v", graph.Function.ReferenceCount)
		}
		if got := graph.NormalFlow.ReadBindings; got != tc.want {
			t.Fatalf("%s: read bindings=%d; want %d", tc.body, got, tc.want)
		}
	}
}

func TestNormalFlowPreservesDistinctReadOccurrences(t *testing.T) {
	source := `package example; func f(x int) int { return x+x }`
	graph := parseIR(t, source, pdg.NewBuilder())
	reads := 0
	for _, node := range graph.Nodes {
		if node.Category != pdg.Read {
			continue
		}
		reads++
		attrs := graph.NodeAttributes(node)
		if len(attrs.Reads) != 1 || attrs.Reads[0].Symbol != graph.Function.Inputs[0].Symbol {
			t.Fatal("read occurrence loses binding")
		}
		span := graph.Spans[node.Span-1]
		if source[span.Start:span.End] != "x" {
			t.Fatal("read span is not the source occurrence")
		}
	}
	if reads != 2 {
		t.Fatalf("read nodes=%d; want 2", reads)
	}
}

func TestScalarNormalProfileRejectsUnlistedSource(t *testing.T) {
	functions := []string{
		`func f() { _="text" }`,
		`func f() { _=1.2 }`,
		`func f() { type Local int; _=1 }`,
		`func f() { const local=1; _=local }`,
		`func f(...int) {}`,
	}
	for _, function := range functions {
		graph := parseIR(t, "package example; "+function, pdg.NewBuilder())
		if graph.NormalFlow != nil {
			t.Fatalf("unsupported function gets scalar normal profile: %s", function)
		}
	}
}

func TestNormalFlowDownstreamRetainsDefReadValuePaths(t *testing.T) {
	source := `package example; func f(x int,c bool) int { if c { x=1 }; return x+x }`
	graph := parseIR(t, source, pdg.NewBuilder())
	view := patterns.MiningView(&graph)
	if len(view.Nodes) != len(graph.Nodes) || len(view.Edges) != len(graph.Edges) {
		t.Fatalf("downstream loses scalar semantics: nodes %d/%d edges %d/%d", len(view.Nodes), len(graph.Nodes), len(view.Edges), len(graph.Edges))
	}
}

func TestNormalFlowMatchingRemovesUnreachableOperations(t *testing.T) {
	source := `package example; func f(x int) int { return x; x=2; return x }`
	graph := parseIR(t, source, pdg.NewBuilder())
	kept := pdg.NormalProjection(&graph)
	removed := 0
	for i, node := range graph.Nodes {
		span := graph.Spans[node.Span-1]
		if source[span.Start:span.End] == "x=2" {
			if kept[i] {
				t.Fatal("unreachable write enters matching")
			}
			removed++
		}
	}
	if removed != 1 {
		t.Fatal("unreachable source operation was erased from the PDG")
	}
	view := patterns.MiningView(&graph)
	if len(view.Nodes) >= len(graph.Nodes) {
		t.Fatal("projection does not remove unreachable operations")
	}
}

func TestScalarNormalNamedResultCountsAsLocalRead(t *testing.T) {
	graph := parseIR(t, `package example; func f() (x int) { x=2; return }`, pdg.NewBuilder())
	if graph.Function.ReferenceCount.Value != 1 {
		t.Fatal("named result read is omitted from reference feature")
	}
	symbol := graph.Function.Outputs[0].Symbol
	if graph.Symbols[symbol-1].Role != pdg.LocalRole {
		t.Fatal("named result does not retain its local binding role")
	}
	if graph.Function.Outputs[0].Role != pdg.ReturnRole {
		t.Fatal("interface loses return role")
	}
}
