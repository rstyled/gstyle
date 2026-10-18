package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

func TestRangeConstantBoundSkipsExpressionEvaluation(t *testing.T) {
	cases := []struct {
		name, source    string
		accesses, calls int
		constant        bool
	}{
		{"array dereference", `func f(p *[3]int){for i:=range *p{_=i}}`, 0, 0, true},
		{"blank value", `func f(p *[3]int){for i,_:=range *p{_=i}}`, 0, 0, true},
		{"no bindings", `func f(p *[3]int){for range *p{}}`, 0, 0, true},
		{"element value", `func f(p *[3]int){for _,v:=range *p{_=v}}`, 1, 0, false},
		{"field selection", `func f(p *struct{xs [3]int}){for i:=range p.xs{_=i}}`, 0, 0, true},
		{"conversion", `func f(p *[3]int){for i:=range (*[3]int)(p){_=i}}`, 0, 0, true},
		{"array literal", `func f(p *int){for i:=range [3]int{*p}{_=i}}`, 0, 0, true},
		{"constant builtin", `func f(p *int){for i:=range [1]int{len([3]int{*p})}{_=i}}`, 0, 0, true},
		{"ordinary call", `func f(read func()int){for i:=range [1]int{read()}{_=i}}`, 0, 1, false},
		{"call then dereference", `func f(get func()*[3]int){for i:=range *get(){_=i}}`, 1, 1, false},
		{"integer constant", `func f(){for i:=range 3{_=i}}`, 0, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, "package p;"+tc.source, pdg.NewBuilder())
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			report, err := runtimeguard.Analyze(&graph)
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Accesses) != tc.accesses || len(report.Calls) != tc.calls {
				t.Fatalf("accesses=%d calls=%d, want %d and %d", len(report.Accesses), len(report.Calls), tc.accesses, tc.calls)
			}
			bound := false
			for _, block := range graph.ValueFlow.Blocks {
				for _, ref := range block.Operations {
					attrs := graph.NodeAttributes(graph.Nodes[ref-1])
					if graph.Tables.Text(attrs.Construct) == "range-bound" {
						bound = true
					}
				}
			}
			if bound != tc.constant {
				t.Fatalf("constant bound=%v, want %v", bound, tc.constant)
			}
		})
	}
}

func TestRangeConstantBoundFeedsDecisionAndIteration(t *testing.T) {
	graph := parseIR(t, `package p;func f(p *[3]int)int{sum:=0;for i:=range *p{sum+=i};return sum}`, pdg.NewBuilder())
	var bound pdg.Ref
	for i, node := range graph.Nodes {
		if graph.Tables.Text(graph.NodeAttributes(node).Construct) == "range-bound" {
			bound = pdg.Ref(i + 1)
		}
	}
	if bound == 0 {
		t.Fatal("missing constant bound")
	}
	decision, iteration := false, false
	for _, edge := range graph.Edges {
		if edge.Source != bound {
			continue
		}
		switch graph.Tables.Text(edge.Subkind) {
		case "control-value":
			decision = true
		case "iteration-value":
			iteration = true
		}
	}
	if !decision || !iteration {
		for _, edge := range graph.Edges {
			t.Logf("bound %d edge %d -> %d %s", bound, edge.Source, edge.Target, graph.Tables.Text(edge.Subkind))
		}
		t.Fatalf("bound dependencies: decision=%v iteration=%v", decision, iteration)
	}
}
