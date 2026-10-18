package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
	"testing"
)

func TestExitArgumentPrecisionRequiresCanonicalInputs(t *testing.T) {
	for _, mode := range []string{"representation", "inventory", "operand"} {
		t.Run(mode, func(t *testing.T) {
			graphs, name := argumentExitGraphs(t, `func f(){stop(false)};func stop(flag bool){if flag{panic("stop")}}`)
			graph := graphs[name]
			attrs := make([]pdg.Attributes, len(graph.Nodes))
			for i, node := range graph.Nodes {
				attrs[i] = graph.NodeAttributes(node)
				if node.Category != pdg.Call {
					continue
				}
				extension := ""
				if mode == "representation" {
					extension = "gstyle.go:input-representation"
				}
				if mode == "inventory" {
					extension = "gstyle.go:value-operands"
				}
				attrs[i].Extensions = slices.DeleteFunc(slices.Clone(attrs[i].Extensions), func(e pdg.Extension) bool { return graph.Tables.Text(e.Name) == extension })
				if mode == "operand" {
					graph.Edges = slices.DeleteFunc(graph.Edges, func(e pdg.Edge) bool { return e.Target == pdg.Ref(i+1) && e.Kind == pdg.Data && e.Position == 1 })
				}
			}
			graph.Attributes = nil
			graph.AttributeTables = pdg.AttributeTables{}
			for i := range graph.Nodes {
				graph.Nodes[i].Attributes = pdg.Ref(i + 1)
			}
			if err := pdg.PackAttributes(graph, attrs); err != nil {
				t.Fatal(err)
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries[name].Ordinary; got != callflow.Returns|callflow.Panics {
				t.Fatalf("missing %s invents precise exit %v", mode, got)
			}
		})
	}
}

func TestExitArgumentTablesAreReadOnlyAfterSolving(t *testing.T) {
	graphs, _ := argumentExitGraphs(t, `func f(){stop(false)};func stop(flag bool){if flag{panic("stop")}}`)
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	fresh, name := argumentExitGraphs(t, `func f(){stop(true)};func stop(flag bool){if flag{panic("stop")}}`)
	graph := fresh[name]
	calls, err := callflow.NewCallSites(graph)
	if err != nil {
		t.Fatal(err)
	}
	var ref pdg.Ref
	for i, node := range graph.Nodes {
		if node.Category == pdg.Call {
			ref = pdg.Ref(i + 1)
		}
	}
	if ref == 0 {
		t.Fatal("missing call")
	}
	for range 3 {
		// This true-input context was not requested while solving the original
		// graphs. A sealed table falls back to its general may-exit summary.
		if got := calls.Outcomes(summaries, ref, false); got != callflow.Returns|callflow.Panics {
			t.Fatalf("new query changes sealed result: %v", got)
		}
	}
}
