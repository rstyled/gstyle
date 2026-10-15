package runtimeguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestSymbolRecursiveChains(t *testing.T) {
	for _, depth := range []int{0, 1, 2, 7, 64, 257} {
		for _, kind := range []string{"cache", "limit", "oversized", "absent", "other", "zero-child", "unavailable-middle", "other-middle", "cached-middle"} {
			t.Run(fmt.Sprintf("%s/%d", kind, depth), func(t *testing.T) {
				view, want := recursiveSymbolFixture(t, depth, kind)
				before := symbolGraphBytes(t, view.graph)
				fields, operands := slices.Clone(view.fields), slices.Clone(view.operands)
				if got := view.symbol(1); got != want {
					t.Fatalf("symbol=%d want=%d", got, want)
				}
				if !bytes.Equal(before, symbolGraphBytes(t, view.graph)) || !slices.Equal(fields, view.fields) || !slices.Equal(operands, view.operands) {
					t.Fatal("recursive lookup changes graph or cache inputs")
				}
			})
		}
	}
}

func recursiveSymbolFixture(t *testing.T, depth int, kind string) (*graphView, pdg.Ref) {
	t.Helper()
	pool := pdg.NewBuilder()
	edge := pool.Text("edge")
	nodes := make([]pdg.Node, depth+2)
	attrs := make([]pdg.Attributes, depth+1)
	view := &graphView{fields: make([]pdg.Ref, len(nodes)), operands: make([][2]pdg.Ref, len(nodes)), kinds: make([]string, 3)}
	for i := 0; i < depth; i++ {
		pool.Tables.Strings = append(pool.Tables.Strings, "parentheses")
		nodes[i] = pdg.Node{Category: pdg.Computation, Attributes: pdg.Ref(i + 1)}
		attrs[i] = pdg.Attributes{Operator: pdg.Fact{Status: pdg.Known, Value: pdg.Text(len(pool.Tables.Strings))}, Position: pdg.Position(i + 1), Outcomes: []pdg.Text{edge, 0, edge}}
		if i%2 == 1 {
			nodes[i].Category = pdg.Read
			attrs[i].Reads = []pdg.Access{{Symbol: 1, Location: 1}}
		}
		view.operands[i] = [2]pdg.Ref{pdg.Ref(i + 2), pdg.Ref(len(nodes))}
	}
	nodes[depth] = pdg.Node{Category: pdg.Read, Attributes: pdg.Ref(depth + 1)}
	nodes[depth+1] = pdg.Node{Category: pdg.Read}
	view.fields[depth+1] = 99
	want := pdg.Ref(0)
	switch kind {
	case "cache", "unavailable-middle", "other-middle", "cached-middle":
		view.fields[depth], want = 7, 7
	case "limit":
		attrs[depth].Reads, want = []pdg.Access{{Symbol: 3}}, 3
	case "oversized":
		attrs[depth].Reads = []pdg.Access{{Symbol: 4}}
	case "absent":
		nodes[depth].Attributes = 0
	case "other":
		nodes[depth].Category = pdg.Computation
		attrs[depth].Operator = pdg.Fact{Status: pdg.Known, Value: pool.Text("conversion")}
	case "zero-child":
		nodes[depth].Category = pdg.Computation
		attrs[depth].Operator = pdg.Fact{Status: pdg.Known, Value: pool.Text("parentheses")}
	default:
		t.Fatalf("unknown fixture %q", kind)
	}
	if depth > 0 {
		middle := depth / 2
		switch kind {
		case "unavailable-middle":
			attrs[middle].Operator.Status, want = pdg.Unknown, 0
		case "other-middle":
			attrs[middle].Operator.Value, want = pool.Text("conversion"), 0
		case "cached-middle":
			view.fields[middle], want = 11, 11
		}
	}
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: nodes}
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
	view.graph = graph
	return view, want
}

func symbolGraphBytes(t *testing.T, graph *pdg.Graph) []byte {
	t.Helper()
	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
