package interfacenil

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestComputationTransferRequiresKnownOperator(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables}
	for _, spelling := range []string{"parentheses", "conversion", "+", ""} {
		for status := 0; status <= 255; status++ {
			attrs := pdg.Attributes{Operator: pdg.Fact{Status: pdg.Status(status), Value: pool.Text(spelling)}}
			want := status == int(pdg.Known) && (spelling == "parentheses" || spelling == "conversion")
			if got := transfer(pdg.Computation, attrs, graph); got != want {
				t.Fatalf("status=%d operator=%q transfer=%t want=%t", status, spelling, got, want)
			}
		}
	}
}

func TestUnavailableOperatorCannotProveInterfaceComparison(t *testing.T) {
	for _, spelling := range []string{"parentheses", "conversion"} {
		for _, status := range []pdg.Status{pdg.Unknown, pdg.Unsupported, pdg.NotApplicable, pdg.Known} {
			graph := interfaceOperatorGraph(t, status, spelling)
			hits := Find([]pdg.Graph{graph})[0]
			want := 0
			if status == pdg.Known {
				want = 1
			}
			if len(hits) != want {
				t.Fatalf("status=%d operator=%q hits=%v want count=%d", status, spelling, hits, want)
			}
		}
	}
}

func interfaceOperatorGraph(t *testing.T, status pdg.Status, spelling string) pdg.Graph {
	t.Helper()
	pool := pdg.NewBuilder()
	evidence := pool.Evidence(pdg.Supported, "test dependency")
	attrs := []pdg.Attributes{
		{Extensions: pool.SingletonExtension("gstyle:interface-value", `"concrete"`)},
		{Operator: pdg.Fact{Status: status, Value: pool.Text(spelling)}},
		{Extensions: pool.SingletonExtension("gstyle:interface-nil-operand", "1")},
	}
	graph := pdg.Graph{Tables: pool.Tables,
		Capabilities: []pdg.NamedCapability{{Name: pool.Text("interface_binding_flow"), Evidence: evidence}},
		Nodes:        []pdg.Node{{Category: pdg.Computation, Attributes: 1}, {Category: pdg.Computation, Attributes: 2}, {Category: pdg.Control, Attributes: 3}},
		Edges: []pdg.Edge{
			{Kind: pdg.Data, Subkind: pool.Text("binding-value"), Evidence: evidence, Source: 1, Target: 2, Position: 1},
			{Kind: pdg.Data, Subkind: pool.Text("binding-value"), Evidence: evidence, Source: 2, Target: 3, Position: 1},
		},
	}
	if err := pdg.PackAttributes(&graph, attrs); err != nil {
		t.Fatal(err)
	}
	return graph
}
