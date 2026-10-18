package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCounterMatchingAbstractionRetainsCanonicalSeed(t *testing.T) {
	source := `package p;func f(xs []string)int{total:=0;for i:=0;i<len(xs);i++{total+=len(xs[i])};return total}`
	graph := parseIR(t, source, pdg.NewBuilder())
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	seed := strings.Index(source, "i:=0") + len("i:=")
	var ref pdg.Ref
	for i, node := range graph.Nodes {
		if node.Span == 0 {
			continue
		}
		span := graph.Spans[node.Span-1]
		if int(span.Start) == seed && int(span.End) == seed+1 {
			ref = pdg.Ref(i + 1)
		}
	}
	if ref == 0 {
		t.Fatal("counter constant lost from canonical PDG")
	}
	if graph.Nodes[ref-1].MatchLabel != 0 {
		t.Fatal("constant counter is not abstracted for matching")
	}
	for _, edge := range graph.Edges {
		if edge.Source == ref && edge.Kind == pdg.Data {
			return
		}
	}
	t.Fatal("counter constant lost its canonical value dependency")
}

func TestCounterMatchingRetainsInitializerEffects(t *testing.T) {
	graph := parseIR(t, `package p;func f(xs []string,start func()int)int{total:=0;for i:=start();i<len(xs);i++{total+=len(xs[i])};return total}`, pdg.NewBuilder())
	view := patterns.MiningView(&graph)
	if got := countKind(view, patterns.Call); got != 2 {
		t.Fatalf("initializer call lost from matching: got %d calls, want 2", got)
	}
}

func TestGeneralMatchingUsesCanonicalReachability(t *testing.T) {
	source := `package p;func f(x string,sink func(string))string{return x;sink(x);return x}`
	graph := parseIR(t, source, pdg.NewBuilder())
	if graph.NormalFlow != nil {
		t.Fatal("general function acquired strict scalar admission")
	}
	retained := pdg.ReachableProjection(&graph)
	if retained == nil {
		t.Fatal("general operation flow is not used by projection")
	}
	for i, node := range graph.Nodes {
		if node.Category == pdg.Call && retained[i] {
			t.Fatal("unreachable sink remains in canonical projection")
		}
	}
	if got := countKind(patterns.MiningView(&graph), patterns.Call); got != 0 {
		t.Fatalf("matching retains %d unreachable calls", got)
	}
}
