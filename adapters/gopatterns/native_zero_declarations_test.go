package gopatterns

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/adapters/govalidation"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestNativeZeroDeclarations(t *testing.T) {
	cases := []string{
		`func f() int32 { var a,b int32; _=a; return b }`,
		`func f() bool { var a,b bool; _=a; return b }`,
		`func f(flag bool) int32 { var a,b int32; if flag { a=1 }; _=b; return a }`,
		`func f(flag bool) int32 { var a,b int32; if flag { return a }; return b }`,
		`func f() int32 { var _,b int32; return b }`,
		`func f() int32 { var a,b,c int32; _=a; _=b; return c }`,
		`func f() { var a,b int32; _=a; _=b }`,
		`func f() int32 { return 1; var a,b int32; _=a; return b }`,
		`func f() int32 { var (a,b int32; c bool); _=a; _=c; return b }`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			pkg := loadTestPackage(t, "package p; "+source)
			root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
			matching, err := ExtractMatching(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if len(matching.Ineligible) != 0 || len(matching.Graphs) != 1 || matching.Graphs[0].Conformance.Status != pdg.Conformant {
				t.Fatalf("zero declaration rejected: %+v", matching.Ineligible)
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

func TestZeroDeclarationsKeepInitializerOrdersExcluded(t *testing.T) {
	for _, source := range []string{
		`func f() int32 { var a,b int32=1,2; _=a; return b }`,
		`func f() int { a,b:=1,2; _=a; return b }`,
		`func f() int32 { var a,b int32; a,b=1,2; _=a; return b }`,
		`func f() int32 { var a,b int32; _=a; return b+1 }`,
		`func f() (int32,int32) { var a,b int32; return a,b }`,
	} {
		t.Run(source, func(t *testing.T) {
			pkg := loadTestPackage(t, "package p; "+source)
			root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
			extraction, err := Extract(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if len(extraction.Callables) != 1 || extraction.Callables[0].Conformance.Status == pdg.Conformant {
				t.Fatal("independent operand/result order is certified")
			}
		})
	}
}

func TestZeroDeclarationHasSeparateDefinitionIdentities(t *testing.T) {
	source := `package p; func f() int32 { var a,b int32; _=a; return b }`
	graph := parseIR(t, source, pdg.NewBuilder())
	fn, fset, info := independentSource(t, source)
	if err := govalidation.Inventory(&graph, fn, fset, info); err != nil {
		t.Fatal(err)
	}
	if err := govalidation.Bindings(&graph, fn, fset, info); err != nil {
		t.Fatal(err)
	}
	if err := govalidation.CFG(&graph, fn, fset); err != nil {
		t.Fatal(err)
	}
	if err := govalidation.Reaching(&graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.Definitions) != 2 || graph.Definitions[0].Node != graph.Definitions[1].Node || graph.Definitions[0].Symbol == graph.Definitions[1].Symbol {
		t.Fatalf("zero bindings lose definition identities: %+v", graph.Definitions)
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
		"value: a,b int32 -> a", "value: a -> _=a", "value: a,b int32 -> b",
		"value: b -> return b", "value: return b -> int32",
		"execution: a,b int32 -> a", "execution: a -> _=a", "execution: _=a -> b", "execution: b -> return b",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) || len(graph.Nodes) != 6 {
		t.Fatalf("unexpected zero declaration facts: nodes=%d\ngot %v\nwant %v", len(graph.Nodes), got, want)
	}
}

func TestZeroDeclarationRejectsMissingOrSwappedDefinitions(t *testing.T) {
	source := `package p; func f() int32 { var a,b int32; _=a; return b }`
	for _, fault := range []string{"missing", "swapped"} {
		t.Run(fault, func(t *testing.T) {
			graph := parseIR(t, source, pdg.NewBuilder())
			changed := false
			for i, edge := range graph.Edges {
				if edge.Definition == 0 {
					continue
				}
				if fault == "missing" {
					graph.Edges = append(graph.Edges[:i], graph.Edges[i+1:]...)
				} else {
					other := pdg.Ref(3) - edge.Definition
					graph.Edges[i].Definition = other
					graph.Edges[i].Symbol = graph.Definitions[other-1].Symbol
				}
				changed = true
				break
			}
			if !changed {
				t.Fatal("no reaching definition found")
			}
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal("corruption must remain structurally valid", err)
			}
			if govalidation.Reaching(&graph) == nil {
				t.Fatal("corrupt definition identity accepted")
			}
		})
	}
}
