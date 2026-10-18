package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestControlProjectionContainsSemanticEdgesOnly(t *testing.T) {
	source := `package p;func f(x int,a,b bool)int{if a{if b{return x}};return 0}`
	graph := parseIR(t, source, pdg.NewBuilder())
	parents := map[pdg.Ref]int{}
	controls := 0
	for _, edge := range graph.Edges {
		if graph.Tables.Text(edge.Subkind) == "lexical-control" {
			t.Fatal("lexical containment is not a dependency")
		}
		if edge.Kind != pdg.ControlEdge {
			continue
		}
		if edge.Position == 0 {
			t.Fatal("semantic outcome missing")
		}
		parents[edge.Target]++
		controls++
	}
	multiple := false
	for _, count := range parents {
		multiple = multiple || count > 1
	}
	if !multiple {
		t.Fatal("multiple semantic controllers lost")
	}
	view := patterns.MiningView(&graph)
	projected := 0
	for _, edge := range view.Edges {
		if edge.Kind != patterns.Ctrl {
			continue
		}
		if edge.ArgPos < 0 {
			t.Fatal("outcome lost")
		}
		projected++
	}
	if projected != controls {
		t.Fatalf("mining discarded semantic edges: %d/%d", projected, controls)
	}
}

func TestUnsupportedControlDoesNotFallBackToLexicalEdges(t *testing.T) {
	for _, body := range []string{`defer work();if a{return};work()`, `go work();if a{work()}`, `recover();if a{work()}`, `ch:=make(chan bool);if <-ch{work()}`} {
		graph := parseIR(t, `package p;func f(a bool){`+body+`};func work(){}`, pdg.NewBuilder())
		for _, edge := range graph.Edges {
			if edge.Kind == pdg.ControlEdge {
				t.Fatalf("unsupported body has asserted control edges: %s", body)
			}
		}
	}
}

func TestSemanticBranchOutcomes(t *testing.T) {
	source := `package p;func f(x int,a bool)int{if a{x=1}else{x=2};return x}`
	graph := parseIR(t, source, pdg.NewBuilder())
	arms := map[string]pdg.Position{}
	for _, edge := range graph.Edges {
		if edge.Kind != pdg.ControlEdge {
			continue
		}
		span := graph.Spans[graph.Nodes[edge.Target-1].Span-1]
		arms[source[span.Start:span.End]] = edge.Position
		want := "true"
		if edge.Position == 2 {
			want = "false"
		}
		if got := graph.Tables.Text(edge.Outcome); got != want {
			t.Fatalf("outcome=%q,want %q", got, want)
		}
	}
	if arms["x=1"] != 1 || arms["x=2"] != 2 {
		t.Fatalf("outcomes=%v", arms)
	}
}
