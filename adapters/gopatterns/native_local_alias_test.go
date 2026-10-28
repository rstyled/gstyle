package gopatterns

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/adapters/govalidation"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestNativeLocalScalarAliases(t *testing.T) {
	cases := []string{
		`func f(a int32) int32 { type local=int32; var x local=a; return x }`,
		`func f() int32 { type local=int32; var x local; return x }`,
		`func f(a bool) bool { type local=bool; var x local=a; return x }`,
		`func f(a int32,b bool) int32 { type local=int32; var x local=a; if b { x=1 }; return x }`,
		`func f(a int32,b bool) int32 { if b { type local=int32; var x local=a; return x }; return a }`,
		`func f(a int32) int32 { type (first=int32; second=first); var x second=a; return x }`,
		`func f(a int32) int32 { type local=int32; { type local=bool; var x local; _=x }; var x local=a; return x }`,
		`func f() { type local=bool; var x local; _=x }`,
		`func f(a uint64) uint64 { type local=uint64; var x local=a; return x }`,
		`func f(a int) int { type local=int; var x local=a; return x }`,
		`func f(a bool) bool { type int32=bool; var x int32=a; return x }`,
		`func f(a int32) int32 { type local=int32; return a }`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			pkg := loadTestPackage(t, "package p; "+source)
			root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
			matching, err := ExtractMatching(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if len(matching.Ineligible) != 0 || len(matching.Graphs) != 1 {
				t.Fatalf("scalar alias rejected: %+v", matching.Ineligible)
			}
			if matching.Graphs[0].Conformance.Status != pdg.Conformant {
				t.Fatal("native graph is not certified")
			}
			extraction, err := Extract(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if len(extraction.Callables) != 1 || extraction.Callables[0].Conformance.Status != pdg.Conformant {
				t.Fatal("routine export disagrees with native admission")
			}
		})
	}
}

func TestLocalAliasAdmissionKeepsUnsupportedDeclarationsPartial(t *testing.T) {
	for _, declaration := range []string{
		`type local int32`,
		`type local=float64`,
		`type local=string`,
		`type local=*int32`,
		`type local=struct{ X int32 }`,
		`type local=func() int32`,
		`type local[T any]=int32`,
	} {
		t.Run(declaration, func(t *testing.T) {
			root := t.TempDir()
			for name, content := range map[string]string{
				"go.mod": "module p\n\ngo 1.26\n",
				"p.go":   "package p; func f(a int32) int32 { " + declaration + "; return a }",
			} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			extraction, err := Extract(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if len(extraction.Callables) != 1 || extraction.Callables[0].Conformance.Status == pdg.Conformant {
				t.Fatal("unsupported declaration is certified")
			}
		})
	}
}

func TestLocalAliasVerifierRejectsCorruptFacts(t *testing.T) {
	source := `package p; func f(a int32) int32 { type local=int32; var x local=a; return x }`
	pkg := loadTestPackage(t, source)
	fn := findFunc(pkg.Syntax[0], "f")
	for _, fault := range []string{"type-operation", "missing-value", "binding-type"} {
		t.Run(fault, func(t *testing.T) {
			pool := pdg.NewBuilder()
			graph, err := buildGraph(pkg, fn, "p.go", pool)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "type-operation":
				alias := fn.Body.List[0].(*ast.DeclStmt).Decl.(*ast.GenDecl).Specs[0]
				graph.Spans = append(graph.Spans, pdg.Span{Source: 1, Start: uint32(pkg.Fset.PositionFor(alias.Pos(), false).Offset), End: uint32(pkg.Fset.PositionFor(alias.End(), false).Offset)})
				fake := graph.Nodes[1]
				fake.ID, fake.Span, fake.OriginalKind = pool.Text("fake-type-operation"), pdg.Ref(len(graph.Spans)), pool.Known("*ast.TypeSpec")
				fake.Category = pdg.Computation
				graph.Nodes = append(graph.Nodes, fake)
			case "missing-value":
				graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool { return graph.Tables.Text(edge.Subkind) == "scalar-value" })
			case "binding-type":
				graph.Symbols[0].Type = pool.Known("bool")
			}
			graph.Tables = pool.Tables
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal("corruption must remain structurally valid", err)
			}
			if validateNativeSource(&graph, pkg, fn) == nil {
				t.Fatal("corrupted alias graph accepted")
			}
		})
	}
}

func TestLocalAliasHasOnlyRuntimeDependencies(t *testing.T) {
	source := `package p; func f(a int32) int32 { type local=int32; var x local=a; return x }`
	graph := parseIR(t, source, pdg.NewBuilder())
	fn, fset, info := independentSource(t, source)
	if err := govalidation.Inventory(&graph, fn, fset, info); err != nil {
		t.Fatal(err)
	}
	if err := govalidation.CFG(&graph, fn, fset); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, edge := range graph.Edges {
		if edge.Kind != pdg.Data && edge.Kind != pdg.Execution {
			t.Fatalf("unexpected dependency kind: %v", edge.Kind)
		}
		kind := "value"
		if edge.Kind == pdg.Execution {
			kind = "execution"
		}
		got = append(got, kind+": "+compatOperationSource(&graph, source, edge.Source)+" -> "+compatOperationSource(&graph, source, edge.Target))
	}
	want := []string{
		"value: a int32 -> a",
		"value: a -> x local=a",
		"value: x local=a -> x",
		"value: x -> return x",
		"value: return x -> int32",
		"execution: a -> x local=a",
		"execution: x local=a -> x",
		"execution: x -> return x",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) || len(graph.Nodes) != 6 {
		t.Fatalf("runtime graph has unexpected facts: nodes=%d\ngot %v\nwant %v", len(graph.Nodes), got, want)
	}
}
