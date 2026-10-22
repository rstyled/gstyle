package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"path/filepath"
	"strings"
	"testing"
)

func TestFreshFieldAssertions(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       int
	}{
		{"fresh pointer field", `box:=&Box{Value:&Target{}};_=box.Value.(*Target)`, 0},
		{"scalar field", `box:=&Box{Value:"safe"};_=box.Value.(string)`, 0},
		{"typed nil field", `box:=&Box{Value:(*Target)(nil)};_=box.Value.(*Target)`, 0},
		{"returned after assertion", `box:=&Box{Value:&Target{}};_=box.Value.(*Target);return box`, 0},
		{"mutate child only", `box:=&Box{Value:&Target{}};box.Value.(*Target).Count=1;return box`, 0},
		{"wrong target", `box:=&Box{Value:&Other{}};_=box.Value.(*Target)`, 1},
		{"nil field", `box:=&Box{Value:nil};_=box.Value.(*Target)`, 1},
		{"missing field", `box:=&Box{};_=box.Value.(*Target)`, 1},
		{"unknown field", `box:=&Box{Value:input};_=box.Value.(*Target)`, 1},
		{"direct overwrite", `box:=&Box{Value:&Target{}};box.Value=&Other{};_=box.Value.(*Target)`, 1},
		{"conditional overwrite", `box:=&Box{Value:&Target{}};if flag{box.Value=&Other{}};_=box.Value.(*Target)`, 1},
		{"alias write", `box:=&Box{Value:&Target{}};alias:=box;alias.Value=&Other{};_=box.Value.(*Target)`, 1},
		{"argument escape", `box:=&Box{Value:&Target{}};change(box);_=box.Value.(*Target)`, 1},
		{"field address", `box:=&Box{Value:&Target{}};replace(&box.Value);_=box.Value.(*Target)`, 1},
		{"captured write", `box:=&Box{Value:&Target{}};func(){box.Value=&Other{}}();_=box.Value.(*Target)`, 1},
		{"address root", `box:=&Box{Value:&Target{}};replaceBox(&box);_=box.Value.(*Target)`, 1},
		{"method call", `box:=&Box{Value:&Target{}};box.change();_=box.Value.(*Target)`, 1},
		{"reassigned root", `box:=&Box{Value:&Target{}};box=&Box{Value:&Other{}};_=box.Value.(*Target)`, 1},
		{"global publication", `box:=&Box{Value:&Target{}};global=box;_=box.Value.(*Target)`, 1},
		{"parameter field", `_=passed.Value.(*Target)`, 1},
		{"factory field", `box:=newBox(input);_=box.Value.(*Target)`, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			end := ";return nil"
			if strings.Contains(test.body, "return box") {
				end = ""
			}
			extraction, _ := proofExtraction(t, `package example
  type Target struct{Count int};type Other struct{};type Box struct{Value any}
  var global *Box
  func change(box *Box){box.Value=&Other{}}
  func(box *Box)change(){box.Value=&Other{}}
  func replace(value *any){*value=&Other{}}
  func replaceBox(box **Box){*box=&Box{Value:&Other{}}}
  func newBox(value any)*Box{return &Box{Value:value}}
  func check(input any,flag bool,passed *Box)*Box{`+test.body+end+`}`)
			for _, fn := range extraction.Funcs {
				if strings.HasSuffix(fn.Name, ".check") {
					if len(fn.ForceTypeAssertHits) != test.want {
						t.Fatalf("got %+v, want %d warnings", fn.ForceTypeAssertHits, test.want)
					}
					return
				}
			}
			t.Fatal("missing check function")
		})
	}
}

func TestFreshFieldRequiresCanonicalEvidence(t *testing.T) {
	for _, missing := range []string{"allocation", "field path", "initializer", "literal input", "assertion input", "flow"} {
		t.Run(missing, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;type Target struct{};type Box struct{Value any};func check(){box:=&Box{Value:&Target{}};_=box.Value.(*Target)}`)
			analysis := packageAnalysis{pkg: pkg, root: filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename), pool: pdg.NewBuilder()}
			if err := analysis.build(); err != nil {
				t.Fatal(err)
			}
			if err := analysis.summarize(); err != nil {
				t.Fatal(err)
			}
			if err := analysis.analyze(); err != nil {
				t.Fatal(err)
			}
			assertNullableWarnings(t, analysis.result, 0)
			changed := false
			for _, source := range analysis.graphs {
				graph := source.graph
				if missing == "flow" {
					graph.NormalFlow, graph.ValueFlow, graph.Extensions = nil, nil, nil
					changed = true
					continue
				}
				if missing == "allocation" {
					for i := range graph.Locations {
						graph.Locations[i].Extensions = nil
						changed = true
					}
					continue
				}
				if missing == "field path" {
					for i, extensions := range graph.AttributeTables.Extensions {
						kept := extensions[:0]
						for _, extension := range extensions {
							if graph.Tables.Text(extension.Name) == "gstyle.go:field-path" {
								changed = true
								continue
							}
							kept = append(kept, extension)
						}
						graph.AttributeTables.Extensions[i] = kept
					}
					continue
				}
				nodes, _ := assertionSources(source.declaration, pkg.Fset, graph)
				kept := graph.Edges[:0]
				for _, edge := range graph.Edges {
					target := nodes[edge.Target]
					_, initializer := target.(*ast.KeyValueExpr)
					_, literal := target.(*ast.CompositeLit)
					_, assertion := target.(*ast.TypeAssertExpr)
					selected := missing == "initializer" && initializer && edge.Position == 2 || missing == "literal input" && literal || missing == "assertion input" && assertion
					if edge.Kind == pdg.Data && selected {
						changed = true
						continue
					}
					kept = append(kept, edge)
				}
				graph.Edges = kept
			}
			if !changed {
				t.Fatal("fixture lacks selected evidence")
			}
			if err := analysis.summarize(); err != nil {
				t.Fatal(err)
			}
			analysis.result = nil
			if err := analysis.analyze(); err != nil {
				t.Fatal(err)
			}
			assertNullableWarnings(t, analysis.result, 1)
		})
	}
}
