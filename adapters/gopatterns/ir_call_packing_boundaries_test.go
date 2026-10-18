package gopatterns

import (
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func mutatePackingFact(t *testing.T, graph *pdg.Graph, name, replacement string) {
	t.Helper()
	attrs := make([]pdg.Attributes, len(graph.Nodes))
	changed := false
	for i, node := range graph.Nodes {
		attrs[i] = graph.NodeAttributes(node)
		var extensions []pdg.Extension
		for _, extension := range attrs[i].Extensions {
			if graph.Tables.Text(extension.Name) == name {
				changed = true
				if replacement == "" {
					continue
				}
				graph.Tables.Strings = append(graph.Tables.Strings, replacement)
				extension.Value = pdg.Text(len(graph.Tables.Strings))
			}
			extensions = append(extensions, extension)
		}
		attrs[i].Extensions = extensions
	}
	if !changed {
		t.Fatal("missing packing fact", name)
	}
	graph.Attributes = nil
	graph.AttributeTables = pdg.AttributeTables{}
	for i := range graph.Nodes {
		graph.Nodes[i].Attributes = pdg.Ref(i + 1)
	}
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
}
func TestCallPackingNeedsCanonicalShapeAndTupleArity(t *testing.T) {
	for _, tc := range []struct{ name, source, fact, replacement string }{
		{"missing pack shape", `func f(){stop()};func stop(values ...int){if values!=nil{panic("bad")}}`, "gstyle.go:call-shape", ""},
		{"missing spread shape", `func f(){var values []int;stop(values...)};func stop(values ...int){if values!=nil{panic("bad")}}`, "gstyle.go:call-shape", ""},
		{"missing tuple arity", `func f(){stop(pair())};func pair()(int,int){return 0,0};func stop(values ...int){if values==nil{panic("bad")}}`, "gstyle.go:tuple-arity", ""},
		{"conflicting tuple arity", `func f(){stop(pair())};func pair()(int,int){return 0,0};func stop(values ...int){if values==nil{panic("bad")}}`, "gstyle.go:tuple-arity", "3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graphs, name := argumentExitGraphs(t, tc.source)
			mutatePackingFact(t, graphs[name], tc.fact, tc.replacement)
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries[name].Ordinary; got != callflow.Returns|callflow.Panics {
				t.Fatalf("missing facts invent precision: %v", got)
			}
		})
	}
}
func TestCallShapeMustAgreeWithSavedDeferContract(t *testing.T) {
	for _, tc := range []struct{ body, shape string }{
		{`var values []int;defer stop(values...)`, `{"fixed":0,"variadic":true,"spread":false,"tuple":0}`},
		{`defer stop(0)`, `{"fixed":0,"variadic":true,"spread":true,"tuple":0}`},
	} {
		graphs, name := argumentExitGraphs(t, `func f(){`+tc.body+`};func stop(values ...int){}`)
		mutatePackingFact(t, graphs[name], "gstyle.go:call-shape", tc.shape)
		if _, err := callflow.ExitEffects(graphs); err == nil || !strings.Contains(err.Error(), "contradicts") {
			t.Fatalf("saved contract conflict accepted: %v", err)
		}
	}
}
func TestDuplicateCallPackingRecordsFailConstruction(t *testing.T) {
	graphs, name := argumentExitGraphs(t, `func f(){stop()};func stop(values ...int){}`)
	graph := graphs[name]
	for i, node := range graph.Nodes {
		if node.Category != pdg.Call {
			continue
		}
		attrs := graph.NodeAttributes(node)
		for _, extension := range attrs.Extensions {
			if graph.Tables.Text(extension.Name) != "gstyle.go:call-shape" {
				continue
			}
			// Packed extension lists are graph-owned. Clone before editing this fixture.
			list := graph.Attributes[node.Attributes-1].Extensions
			graph.AttributeTables.Extensions[list-1] = append(slices.Clone(attrs.Extensions), extension)
			if _, err := callflow.ExitEffects(graphs); err == nil {
				t.Fatalf("node %d duplicate accepted", i+1)
			}
			return
		}
	}
	t.Fatal("missing call shape")
}

func TestCallPackingRequiresContiguousExpressionSlots(t *testing.T) {
	graphs, name := argumentExitGraphs(t, `func f(){stop(0)};func stop(values ...int){if values==nil{panic("bad")}}`)
	graph := graphs[name]
	attrs := make([]pdg.Attributes, len(graph.Nodes))
	for i, node := range graph.Nodes {
		attrs[i] = graph.NodeAttributes(node)
		if node.Category != pdg.Call {
			continue
		}
		extensions := slices.Clone(attrs[i].Extensions)
		for j, extension := range extensions {
			if graph.Tables.Text(extension.Name) != "gstyle.go:value-operands" {
				continue
			}
			graph.Tables.Strings = append(graph.Tables.Strings, "[2]")
			extensions[j].Value = pdg.Text(len(graph.Tables.Strings))
		}
		attrs[i].Extensions = extensions
		for j, edge := range graph.Edges {
			if edge.Target == pdg.Ref(i+1) && edge.Kind == pdg.Data && edge.Position == 1 {
				graph.Edges[j].Position = 2
			}
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
		t.Fatalf("noncontiguous slots invent pack arity: %v", got)
	}
}
