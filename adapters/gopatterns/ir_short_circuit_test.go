package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestShortCircuitNormalFlowFixtures(t *testing.T) {
	cases := []string{
		`return a && b`,
		`return a || b`,
		`return a && (b || c)`,
		`if a && b { return c }; return a`,
		`x:=a || b; return x && c`,
		`return false && b`,
		`return true || b`,
	}
	for _, body := range cases {
		t.Run(body, func(t *testing.T) {
			source := "package example; func f(a,b,c bool) bool { " + body + " }"
			graph := parseIR(t, source, pdg.NewBuilder())
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			if graph.NormalFlow == nil {
				t.Fatal("short circuit lacks normal-flow context")
			}
			predicates, joins := 0, 0
			for i, node := range graph.Nodes {
				attrs := graph.NodeAttributes(node)
				if node.Category == pdg.Control && graph.Tables.Text(attrs.Construct) == "short-circuit" {
					predicates++
					hasValue := false
					for _, edge := range graph.Edges {
						if edge.Target == pdg.Ref(i+1) && edge.Kind == pdg.Data {
							hasValue = true
						}
					}
					if !hasValue {
						t.Fatal("short-circuit predicate loses tested value")
					}
				}
				if node.Category == pdg.Merge {
					joins++
				}
			}
			if predicates == 0 || joins < predicates {
				t.Fatal("logical expression is not branched and joined")
			}
		})
	}
}

func TestShortCircuitRHSIsConditionalAndNotRequiredAtJoin(t *testing.T) {
	for _, operator := range []string{"&&", "||"} {
		source := "package example; func f(a,b bool) bool { return a " + operator + " b }"
		graph := parseIR(t, source, pdg.NewBuilder())
		var predicate, rhs, ret pdg.Ref
		for i, node := range graph.Nodes {
			attrs := graph.NodeAttributes(node)
			span := graph.Spans[node.Span-1]
			if node.Category == pdg.Control && graph.Tables.Text(attrs.Construct) == "short-circuit" {
				predicate = pdg.Ref(i + 1)
			}
			if node.Category == pdg.Read && source[span.Start:span.End] == "b" {
				rhs = pdg.Ref(i + 1)
			}
			if node.Category == pdg.Return {
				ret = pdg.Ref(i + 1)
			}
		}
		conditional, joined := false, false
		for _, edge := range graph.Edges {
			if edge.Kind == pdg.ControlEdge && edge.Source == predicate && edge.Target == rhs {
				conditional = true
			}
			if edge.Kind == pdg.Execution && edge.Source == predicate && edge.Target == ret {
				joined = true
			}
			if edge.Kind == pdg.Execution && edge.Source == rhs && edge.Target == ret {
				t.Fatal("skipped RHS becomes mandatory execution prerequisite")
			}
		}
		if !conditional || !joined {
			t.Fatalf("%s: conditional RHS=%t join prerequisite=%t", operator, conditional, joined)
		}
	}
}
