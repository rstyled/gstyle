package runtimeguard

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestDirectBindingReadAdmission(t *testing.T) {
	entries := []pdg.Access{{}, {Symbol: 1}, {Symbol: 0xffffffff}, {Location: 1}, {Location: 0xffffffff}, {Symbol: 1, UnresolvedReason: 9, Position: 7}}
	for category := 0; category < 256; category++ {
		node := pdg.Node{Category: pdg.Category(category)}
		for _, entry := range entries {
			for count := 0; count <= 2; count++ {
				reads := make([]pdg.Access, count)
				for i := range reads {
					reads[i] = entry
				}
				attrs := pdg.Attributes{Reads: reads}
				before := append([]pdg.Access{}, reads...)
				want := node.Category == pdg.Read && count == 1 && entry.Location == 0
				if got := directBindingRead(node, attrs); got != want {
					t.Fatalf("category=%d count=%d entry=%+v: got=%t want=%t", category, count, entry, got, want)
				}
				if !reflect.DeepEqual(reads, before) {
					t.Fatal("direct read changes input records")
				}
			}
		}
	}
}

func TestSymbolZeroAndCache(t *testing.T) {
	var absent *graphView
	if got := absent.symbol(0); got != 0 {
		t.Fatalf("symbol(0)=%d", got)
	}
	view := &graphView{fields: []pdg.Ref{0xffffffff}}
	if got := view.symbol(1); got != 0xffffffff {
		t.Fatalf("cached symbol=%d", got)
	}
	if view.fields[0] != 0xffffffff {
		t.Fatal("symbol lookup changes cache")
	}
}

func TestSymbolDirectReadAdmission(t *testing.T) {
	for _, category := range []pdg.Category{pdg.Read, pdg.Assignment, pdg.Computation} {
		for _, symbol := range []pdg.Ref{0, 1, 2, 3, 0xffffffff} {
			for _, location := range []pdg.Ref{0, 1, 0xffffffff} {
				for count := 0; count <= 2; count++ {
					pool := pdg.NewBuilder()
					reads := make([]pdg.Access, count)
					for i := range reads {
						reads[i] = pdg.Access{Symbol: symbol, Location: location}
					}
					graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: category, Attributes: 1}}}
					if err := pdg.PackAttributes(&graph, []pdg.Attributes{{Reads: reads}}); err != nil {
						t.Fatal(err)
					}
					view := &graphView{graph: &graph, fields: []pdg.Ref{0}, kinds: []string{`"signed"`, `"signed"`}}
					want := pdg.Ref(0)
					if category == pdg.Read && count == 1 && location == 0 && symbol <= 2 {
						want = symbol
					}
					if got := view.symbol(1); got != want {
						t.Fatalf("category=%d symbol=%d location=%d count=%d: got=%d want=%d", category, symbol, location, count, got, want)
					}
					if !slices.Equal(graph.NodeAttributes(graph.Nodes[0]).Reads, reads) {
						t.Fatal("symbol lookup changes read records")
					}
					if view.fields[0] != 0 {
						t.Fatal("symbol lookup changes cache")
					}
				}
			}
		}
	}
}

func TestSymbolOperatorStatus(t *testing.T) {
	for status := 0; status < 256; status++ {
		pool := pdg.NewBuilder()
		graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Computation, Attributes: 1}, {Category: pdg.Read}}}
		attrs := []pdg.Attributes{{Operator: pdg.Fact{Status: pdg.Status(status), Value: pool.Text("parentheses")}}}
		if err := pdg.PackAttributes(&graph, attrs); err != nil {
			t.Fatal(err)
		}
		view := &graphView{graph: &graph, fields: []pdg.Ref{0, 7}, operands: [][2]pdg.Ref{{2, 0}, {0, 0}}}
		want := pdg.Ref(0)
		if pdg.Status(status) == pdg.Known {
			want = 7
		}
		if got := view.symbol(1); got != want {
			t.Fatalf("status=%d: got=%d want=%d", status, got, want)
		}
	}
}

func TestSymbolKnownOperatorNames(t *testing.T) {
	for _, name := range []string{"", "parentheses", "Parentheses", "parentheses-more", "parentheses\x00", "conversion", "\xff"} {
		pool := pdg.NewBuilder()
		graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Computation, Attributes: 1}, {Category: pdg.Read}}}
		attrs := []pdg.Attributes{{Operator: pdg.Fact{Status: pdg.Known, Value: pool.Text(name)}}}
		if err := pdg.PackAttributes(&graph, attrs); err != nil {
			t.Fatal(err)
		}
		view := &graphView{graph: &graph, fields: []pdg.Ref{0, 7}, operands: [][2]pdg.Ref{{2, 0}, {0, 0}}}
		want := pdg.Ref(0)
		if name == "parentheses" {
			want = 7
		}
		if got := view.symbol(1); got != want {
			t.Fatalf("operator=%q: got=%d want=%d", name, got, want)
		}
	}
}

func TestSymbolParenthesesZeroChild(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Computation, Attributes: 1}}}
	attrs := []pdg.Attributes{{Operator: pdg.Fact{Status: pdg.Known, Value: pool.Text("parentheses")}}}
	if err := pdg.PackAttributes(&graph, attrs); err != nil {
		t.Fatal(err)
	}
	view := &graphView{graph: &graph, fields: []pdg.Ref{0}, operands: [][2]pdg.Ref{{0, 0}}}
	if got := view.symbol(1); got != 0 {
		t.Fatalf("zero-child symbol=%d", got)
	}
}

func TestSymbolAbsentAttributesWithoutTables(t *testing.T) {
	for _, records := range [][]pdg.AttributeSet{nil, {}, {{}}} {
		for _, attribute := range []pdg.Ref{0, 1, 2, 0xffffffff} {
			graph := pdg.Graph{Nodes: []pdg.Node{{Category: pdg.Read, Attributes: attribute}}, Attributes: records}
			view := &graphView{graph: &graph, fields: []pdg.Ref{0}}
			if got := view.symbol(1); got != 0 {
				t.Fatalf("attributes=%d records=%d: symbol=%d", attribute, len(records), got)
			}
		}
	}
}
