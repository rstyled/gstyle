package gopatterns

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestNormalFlowControlFixtures(t *testing.T) {
	cases := []struct {
		name, body string
		want       []string
	}{
		{"join", `if c { x = 1 }; return x`, []string{`if c { x = 1 } -> 1`, `if c { x = 1 } -> x = 1`}},
		{"early-return", `if c { return 1 }; return x`, []string{`if c { return 1 } -> 1`, `if c { return 1 } -> return 1`, `if c { return 1 } -> return x`, `if c { return 1 } -> x`}},
		{"two-arms", `if c { x = 1 } else { x = 2 }; return x`, []string{`if c { x = 1 } else { x = 2 } -> 1`, `if c { x = 1 } else { x = 2 } -> x = 1`, `if c { x = 1 } else { x = 2 } -> 2`, `if c { x = 1 } else { x = 2 } -> x = 2`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package example; func f(x int,c bool) int { " + tc.body + " }"
			graph := parseIR(t, source, pdg.NewBuilder())
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			got := normalEdgeDescriptions(graph, source, pdg.ControlEdge)
			slices.Sort(tc.want)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v; want %v", got, tc.want)
			}
			for _, edge := range graph.Edges {
				if edge.Kind == pdg.ControlEdge && graph.Tables.Evidence[edge.Evidence-1].Status != pdg.Supported {
					t.Fatal("approximate control edge retained")
				}
			}
		})
	}
}

func normalEdgeDescriptions(graph pdg.Graph, source string, kind pdg.EdgeKind) []string {
	var result []string
	for _, edge := range graph.Edges {
		if edge.Kind != kind {
			continue
		}
		from := graph.Spans[graph.Nodes[edge.Source-1].Span-1]
		to := graph.Spans[graph.Nodes[edge.Target-1].Span-1]
		result = append(result, source[from.Start:from.End]+" -> "+source[to.Start:to.End])
	}
	slices.Sort(result)
	return result
}

func TestNormalFlowExecutionIncludesExpressionOperations(t *testing.T) {
	source := `package example; func f(x int) int { x = 1 + 2; return x }`
	graph := parseIR(t, source, pdg.NewBuilder())
	want := []string{"1 -> 2", "2 -> 1 + 2", "1 + 2 -> x = 1 + 2", "x = 1 + 2 -> x", "x -> return x"}
	slices.Sort(want)
	if got := normalEdgeDescriptions(graph, source, pdg.Execution); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v; want %v", got, want)
	}
}

func TestNormalFlowUnsupportedConstructsKeepPartialEvidence(t *testing.T) {
	for _, body := range []string{`for c { x++ }; return x`, `x=g(x); return x`} {
		graph := parseIR(t, "package example; func f(x int,c bool) int { "+body+" }; func g(x int) int { return x }", pdg.NewBuilder())
		for _, capability := range graph.Capabilities {
			name := graph.Tables.Text(capability.Name)
			if (name == "control_dependencies" || name == "execution_dependencies") && graph.Tables.Evidence[capability.Evidence-1].Status == pdg.Supported {
				t.Fatal("unsupported source claims complete normal flow")
			}
		}
	}
}
