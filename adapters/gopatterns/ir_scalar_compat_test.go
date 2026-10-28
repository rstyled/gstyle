package gopatterns

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/adapters/govalidation"
	"github.com/rstyled/gstyle/domain/pdg"
)

func compatOperationSource(g *pdg.Graph, source string, ref pdg.Ref) string {
	span := g.Spans[g.Nodes[ref-1].Span-1]
	return source[span.Start:span.End]
}

func TestScalarBranchReadsOriginalDefinitions(t *testing.T) {
	for _, tail := range []string{"return x", "y:=x; _=y; return x"} {
		source := "package p; func f(a int32,b bool) int32 { x:=a; if b { x=1 } else { x=2 }; " + tail + " }"
		g := parseIR(t, source, pdg.NewBuilder())
		if err := govalidation.Reaching(&g); err != nil {
			t.Fatal(err)
		}
		var reads []pdg.Ref
		for i, node := range g.Nodes {
			if node.Category == pdg.Merge {
				t.Fatal("binding read introduced a synthetic definition")
			}
			if node.Category == pdg.Read && compatOperationSource(&g, source, pdg.Ref(i+1)) == "x" {
				reads = append(reads, pdg.Ref(i+1))
			}
		}
		if len(reads) != 1 && tail == "return x" || len(reads) != 2 && tail != "return x" {
			t.Fatalf("unexpected read sites: %v", reads)
		}
		for _, read := range reads {
			var got []string
			for _, edge := range g.Edges {
				if edge.Target != read {
					continue
				}
				if edge.Kind == pdg.ControlEdge {
					t.Fatal("post-join read is controlled by a branch outcome")
				}
				if edge.Kind == pdg.Execution && (compatOperationSource(&g, source, edge.Source) == "x=1" || compatOperationSource(&g, source, edge.Source) == "x=2") {
					t.Fatal("individual arm write dominates joined read")
				}
				if edge.Kind != pdg.Data {
					continue
				}
				if g.Tables.Text(edge.Subkind) != "reaching-definition" || edge.Definition == 0 {
					t.Fatalf("unexpected read dependency: %+v", edge)
				}
				d := g.Definitions[edge.Definition-1]
				access := g.NodeAttributes(g.Nodes[read-1]).Reads[0]
				if d.Node != edge.Source || d.Symbol != edge.Symbol || edge.Symbol != access.Symbol || edge.Position != access.Position {
					t.Fatal("reaching-definition identity/slot lost")
				}
				got = append(got, compatOperationSource(&g, source, edge.Source))
			}
			slices.Sort(got)
			if !reflect.DeepEqual(got, []string{"x=1", "x=2"}) {
				t.Fatalf("original reaching writes: got %v", got)
			}
		}
	}
}

func TestScalarShortCircuitSelectedValues(t *testing.T) {
	tests := []struct {
		expression string
		want       []string
	}{
		{"a && b", []string{"a -> a && b : a false 2", "b -> a && b : a true 1"}},
		{"a || b", []string{"a -> a || b : a true 1", "b -> a || b : a false 2"}},
		{"a && (b || c)", []string{"a -> a && (b || c) : a false 2", "(b || c) -> a && (b || c) : a true 1", "b -> b || c : b true 1", "c -> b || c : b false 2"}},
		{"(a && b) || c", []string{"a -> a && b : a false 2", "b -> a && b : a true 1", "(a && b) -> (a && b) || c : (a && b) true 1", "c -> (a && b) || c : (a && b) false 2"}},
		{"a && a", []string{"a -> a && a : a false 2", "a -> a && a : a true 1"}},
	}
	for _, tc := range tests {
		t.Run(tc.expression, func(t *testing.T) {
			source := "package p; func f(a,b,c bool) bool { return " + tc.expression + " }"
			g := parseIR(t, source, pdg.NewBuilder())
			fn, fset, info := independentSource(t, source)
			if err := govalidation.Values(&g, fn, fset, info); err != nil {
				t.Fatal(err)
			}
			ids := map[string]pdg.Ref{}
			for i, n := range g.Nodes {
				ids[g.Tables.Text(n.ID)] = pdg.Ref(i + 1)
			}
			var got []string
			for _, edge := range g.Edges {
				if edge.Kind != pdg.Data || g.Nodes[edge.Target-1].Category != pdg.Merge {
					continue
				}
				if g.Tables.Text(edge.Subkind) != "value_alternative" {
					t.Fatalf("eager operand at selected-value join: %+v", edge)
				}
				if edge.Annotations == 0 {
					t.Fatal("alternative lacks controller")
				}
				extensions := g.Annotations[edge.Annotations-1]
				if len(extensions) != 1 || g.Tables.Text(extensions[0].Name) != "gstyle.go:guards" {
					t.Fatal("unexpected guard encoding")
				}
				var guards [][]map[string]string
				if err := json.Unmarshal([]byte(g.Tables.Text(extensions[0].Value)), &guards); err != nil {
					t.Fatal(err)
				}
				if len(guards) != 1 || len(guards[0]) != 1 {
					t.Fatal("expected local decision guard")
				}
				guard := guards[0][0]
				predicate := ids[guard["controller_pdg_node_id"]]
				if predicate == 0 || g.Nodes[predicate-1].Category != pdg.Control || guard["outcome"] != g.Tables.Text(edge.Outcome) {
					t.Fatal("alternative controller/outcome disagrees")
				}
				got = append(got, fmt.Sprintf("%s -> %s : %s %s %d", compatOperationSource(&g, source, edge.Source), compatOperationSource(&g, source, edge.Target), compatOperationSource(&g, source, predicate), guard["outcome"], edge.Position))
			}
			slices.Sort(got)
			slices.Sort(tc.want)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestScalarVerifierRejectsCorruptedAlternatives(t *testing.T) {
	source := "package p; func f(a,b bool) bool { return a || b }"
	for _, corruption := range []string{"missing", "duplicate", "reverse", "position", "outcome", "controller", "unguarded", "eager"} {
		t.Run(corruption, func(t *testing.T) {
			pool := pdg.NewBuilder()
			g := parseIR(t, source, pool)
			fn, fset, info := independentSource(t, source)
			for i, edge := range g.Edges {
				if g.Tables.Text(edge.Subkind) != "value_alternative" {
					continue
				}
				switch corruption {
				case "missing":
					g.Edges = append(g.Edges[:i], g.Edges[i+1:]...)
				case "duplicate":
					edge.ID = pool.Text("duplicate")
					g.Edges = append(g.Edges, edge)
				case "reverse":
					g.Edges[i].Source, g.Edges[i].Target = edge.Target, edge.Source
				case "position":
					g.Edges[i].Position = 3 - edge.Position
				case "outcome":
					g.Edges[i].Outcome = pool.Text("false")
				case "controller":
					raw := `[[{"controller_pdg_node_id":"` + g.Tables.Text(g.Nodes[edge.Source-1].ID) + `","outcome":"true"}]]`
					g.Annotations[edge.Annotations-1][0].Value = pool.Text(raw)
				case "unguarded":
					g.Edges[i].Annotations = 0
				case "eager":
					g.Edges[i].Subkind = pool.Text("scalar-value")
				}
				break
			}
			g.Tables = pool.Tables
			if err := pdg.Validate(&g); err != nil {
				t.Fatal("corruption must remain structurally valid", err)
			}
			if govalidation.Values(&g, fn, fset, info) == nil {
				t.Fatal("corrupted alternative accepted")
			}
		})
	}
}

func TestScalarVerifierRejectsCorruptedReachingSet(t *testing.T) {
	source := "package p; func f(a int32,b bool) int32 { x:=a; if b { x=1 } else { x=2 }; return x }"
	for _, corruption := range []string{"missing", "duplicate", "killed"} {
		t.Run(corruption, func(t *testing.T) {
			pool := pdg.NewBuilder()
			g := parseIR(t, source, pool)
			for i, edge := range g.Edges {
				if edge.Definition == 0 || compatOperationSource(&g, source, edge.Source) != "x=1" {
					continue
				}
				switch corruption {
				case "missing":
					g.Edges = append(g.Edges[:i], g.Edges[i+1:]...)
				case "duplicate":
					edge.ID = pool.Text("duplicate")
					g.Edges = append(g.Edges, edge)
				case "killed":
					for d, definition := range g.Definitions {
						if compatOperationSource(&g, source, definition.Node) == "x:=a" {
							g.Edges[i].Source = definition.Node
							g.Edges[i].Definition = pdg.Ref(d + 1)
						}
					}
				}
				break
			}
			g.Tables = pool.Tables
			if err := pdg.Validate(&g); err != nil {
				t.Fatal("corruption must remain structurally valid", err)
			}
			if govalidation.Reaching(&g) == nil {
				t.Fatal("corrupted reaching set accepted")
			}
		})
	}
}
