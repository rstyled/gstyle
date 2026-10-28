package gopatterns

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/adapters/govalidation"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestNativeVoidReturnPaths(t *testing.T) {
	cases := []struct {
		name, function string
		implicit       int
	}{
		{"empty", `func f() {}`, 1},
		{"local work", `func f(x int32) { x = -x }`, 1},
		{"explicit", `func f() { return }`, 0},
		{"nested block", `func f() { { return } }`, 0},
		{"conditional", `func f(b bool) { if b { return } }`, 1},
		{"early then final return", `func f(b bool) { if b { return }; return }`, 0},
		{"both return", `func f(b bool) { if b { return } else { return } }`, 0},
		{"unreachable suffix", `func f(x int32) { return; x = 1 }`, 0},
		{"empty arms", `func f(b bool) { if b {} else {} }`, 1},
		{"nested branches", `func f(a,b bool) { if a { if b { return }; return } }`, 1},
		{"short circuit", `func f(a,b bool) { if a && b { return } }`, 1},
		{"else if", `func f(a,b bool) { if a { return } else if b { return } }`, 1},
		{"if initializer", `func f(a,b bool) { if x:=a; b { _=x; return } }`, 1},
		{"constant condition", `func f() { if true { return } }`, 1},
		{"UTF-8 offsets", `func f(条件 bool) { if 条件 { return } }`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, "package p; "+tc.function)
			root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
			matching, err := ExtractMatching(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if len(matching.Ineligible) != 0 || len(matching.Graphs) != 1 {
				t.Fatalf("void source rejected: %+v", matching.Ineligible)
			}
			graph := matching.Graphs[0]
			if graph.Conformance.Status != pdg.Conformant || len(graph.Function.Outputs) != 0 {
				t.Fatal("void interface is not certified")
			}
			implicit := 0
			for _, node := range graph.Nodes {
				if node.Category == pdg.Return && graph.Tables.Text(graph.NodeAttributes(node).Construct) == "implicit-return" {
					implicit++
				}
			}
			if implicit != tc.implicit {
				t.Fatalf("implicit returns=%d; want %d", implicit, tc.implicit)
			}
			extraction, err := Extract(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if len(extraction.Callables) != 1 || extraction.Callables[0].Conformance.Status != pdg.Conformant {
				t.Fatal("routine extraction disagrees with native admission")
			}
		})
	}
}

func TestVoidVerifierRejectsCorruptReturnAndBranchFacts(t *testing.T) {
	pkg := loadTestPackage(t, "package p; func f(b bool) { if b { return } }")
	fn := findFunc(pkg.Syntax[0], "f")
	for _, fault := range []string{"span", "construct", "exit", "explicit-exit", "source-kind", "duplicate", "detached", "spurious-value", "missing-order", "outcome", "position", "predicate-outcomes", "reversed-outcomes"} {
		t.Run(fault, func(t *testing.T) {
			pool := pdg.NewBuilder()
			graph, err := buildGraph(pkg, fn, "p.go", pool)
			if err != nil {
				t.Fatal(err)
			}
			ref := implicitReturnRef(t, &graph)
			node := &graph.Nodes[ref-1]
			switch fault {
			case "span":
				graph.Spans[node.Span-1].Start--
				graph.Spans[node.Span-1].End--
			case "construct":
				graph.Attributes[node.Attributes-1].Construct = pool.Text("unverified-return")
			case "exit":
				graph.Attributes[node.Attributes-1].ExitKind = pdg.ExceptionalExit
			case "explicit-exit":
				for i, operation := range graph.Nodes {
					if operation.Category == pdg.Return && pdg.Ref(i+1) != ref {
						graph.Attributes[operation.Attributes-1].ExitKind = pdg.ExceptionalExit
					}
				}
			case "source-kind":
				node.OriginalKind = pool.Known("*ast.ReturnStmt")
			case "duplicate":
				duplicate := *node
				duplicate.ID = pool.Text("duplicate-return")
				graph.Nodes = append(graph.Nodes, duplicate)
			case "detached":
				for i := range graph.NormalFlow.Blocks {
					graph.NormalFlow.Blocks[i].Operations = slices.DeleteFunc(graph.NormalFlow.Blocks[i].Operations, func(operation pdg.Ref) bool { return operation == ref })
				}
			case "spurious-value":
				edge := graph.Edges[0]
				edge.ID, edge.Kind, edge.Target = pool.Text("spurious-value"), pdg.Data, ref
				edge.Definition, edge.Symbol, edge.Outcome = 0, 0, 0
				edge.Position, edge.Subkind = 1, pool.Text("scalar-value")
				graph.Edges = append(graph.Edges, edge)
			case "predicate-outcomes", "reversed-outcomes":
				for _, operation := range graph.Nodes {
					if operation.Category == pdg.Control {
						attrs := &graph.Attributes[operation.Attributes-1]
						if fault == "predicate-outcomes" {
							attrs.Outcomes = 0
							continue
						}
						graph.AttributeTables.References = append(graph.AttributeTables.References, []pdg.Ref{pdg.Ref(pool.Text("false")), pdg.Ref(pool.Text("true"))})
						attrs.Outcomes = pdg.Ref(len(graph.AttributeTables.References))
					}
				}
			default:
				for i, edge := range graph.Edges {
					if edge.Kind != pdg.ControlEdge || edge.Target != ref {
						continue
					}
					switch fault {
					case "missing-order":
						graph.Edges = append(graph.Edges[:i], graph.Edges[i+1:]...)
					case "outcome":
						graph.Edges[i].Outcome = pool.Text("true")
					case "position":
						graph.Edges[i].Position = 1
					}
					break
				}
			}
			graph.Tables = pool.Tables
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal("fault must stay structurally valid", err)
			}
			err = govalidation.Inventory(&graph, fn, pkg.Fset, pkg.TypesInfo)
			if err == nil {
				err = govalidation.CFG(&graph, fn, pkg.Fset)
			}
			if err == nil {
				err = govalidation.Ordering(&graph)
			}
			if err == nil {
				err = govalidation.Values(&graph, fn, pkg.Fset, pkg.TypesInfo)
			}
			if err == nil {
				t.Fatal("corrupt semantic facts passed independent validation")
			}
		})
	}
}

func TestVoidBranchHasExactNormalDependencies(t *testing.T) {
	graph := parseIR(t, "package p; func f(b bool) { if b { return } }", pdg.NewBuilder())
	label := func(ref pdg.Ref) string {
		node := graph.Nodes[ref-1]
		if node.Category == pdg.Return {
			if graph.Tables.Text(graph.NodeAttributes(node).Construct) == "implicit-return" {
				return "fallthrough"
			}
			return "return"
		}
		return map[pdg.Category]string{pdg.FormalInput: "input", pdg.Read: "read", pdg.Control: "if"}[node.Category]
	}
	var got []string
	for _, edge := range graph.Edges {
		source, target := label(edge.Source), label(edge.Target)
		switch edge.Kind {
		case pdg.Execution:
			got = append(got, fmt.Sprintf("execution %s -> %s", source, target))
		case pdg.ControlEdge:
			got = append(got, fmt.Sprintf("control %s -> %s [%d %s]", source, target, edge.Position, graph.Tables.Text(edge.Outcome)))
		case pdg.Data:
			if edge.Definition != 0 {
				got = append(got, fmt.Sprintf("data %s -> %s [definition]", source, target))
				continue
			}
			got = append(got, fmt.Sprintf("data %s -> %s [operand %d]", source, target, edge.Position))
		}
	}
	want := []string{
		"data input -> read [definition]",
		"data read -> if [operand 1]",
		"execution read -> if",
		"execution if -> return",
		"execution if -> fallthrough",
		"control if -> return [1 true]",
		"control if -> fallthrough [2 false]",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normal dependencies:\ngot %v\nwant %v", got, want)
	}
}

func implicitReturnRef(t *testing.T, graph *pdg.Graph) pdg.Ref {
	t.Helper()
	for i, node := range graph.Nodes {
		if node.Category == pdg.Return && graph.Tables.Text(graph.NodeAttributes(node).Construct) == "implicit-return" {
			return pdg.Ref(i + 1)
		}
	}
	t.Fatal("implicit return is missing")
	return 0
}
