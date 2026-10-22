package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateHelperAssertions(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         int
	}{
		{"guarded caller", `func leaf(v any){_=v.(string)};func caller(v any){if _,ok:=v.(string);ok{leaf(v)}}`, 0},
		{"concrete caller", `func leaf(v any){_=v.(string)};func caller(){leaf("safe")}`, 0},
		{"returned guard", `func leaf(v any){_=v.(string)};func caller(v any){if _,ok:=v.(string);!ok{return};leaf(v)}`, 0},
		{"type case caller", `func leaf(v any){_=v.(string)};func caller(v any){switch v.(type){case string:leaf(v)}}`, 0},
		{"alias parameter", `func leaf(v any){copy:=v;_=copy.(string)};func caller(){leaf("safe")}`, 0},
		{"unguarded caller", `func leaf(v any){_=v.(string)};func caller(v any){leaf(v)}`, 1},
		{"mixed callers", `func leaf(v any){_=v.(string)};func good(){leaf("safe")};func bad(v any){leaf(v)}`, 1},
		{"wrong caller type", `func leaf(v any){_=v.(string)};func caller(){leaf(7)}`, 1},
		{"nil caller", `func leaf(v any){_=v.(string)};func caller(){leaf(nil)}`, 1},
		{"function escape", `func leaf(v any){_=v.(string)};var escaped=leaf;func caller(){leaf("safe")}`, 1},
		{"no caller", `func leaf(v any){_=v.(string)}`, 1},
		{"exported function", `func Leaf(v any){_=v.(string)};func caller(){Leaf("safe")}`, 1},
		{"method", `type box struct{};func(box)leaf(v any){_=v.(string)};func caller(){box{}.leaf("safe")}`, 1},
		{"overwrite parameter", `func leaf(v any){v=7;_=v.(string)};func caller(){leaf("safe")}`, 1},
		{"conditional overwrite", `func leaf(v any,flag bool){if flag{v=7};_=v.(string)};func caller(flag bool){leaf("safe",flag)}`, 1},
		{"captured parameter", `func leaf(v any){func(){v=7}();_=v.(string)};func caller(){leaf("safe")}`, 1},
		{"escaped parameter", `func leaf(v any){replace(&v);_=v.(string)};func caller(){leaf("safe")};func replace(p *any){*p=7}`, 1},
		{"wrong guard", `func leaf(v any){_=v.(string)};func caller(v,other any){if _,ok:=other.(string);ok{leaf(v)}}`, 1},
		{"stale guard", `func leaf(v any){_=v.(string)};func caller(v any){if _,ok:=v.(string);!ok{return};v=7;leaf(v)}`, 1},
		{"guard bypass", `func leaf(v any){_=v.(string)};func caller(v any,flag bool){if flag{goto use};if _,ok:=v.(string);!ok{return};use:leaf(v)}`, 1},
		{"recursion", `func leaf(v any){_=v.(string);leaf(v)};func caller(){leaf("safe")}`, 1},
		{"generic", `func leaf[T any](v any){_=v.(string)};func caller(){leaf[int]("safe")}`, 1},
		{"closure caller", `func leaf(v any){_=v.(string)};func caller(){func(){leaf("safe")}()}`, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, "package example\n"+test.source)
			seen := false
			for _, fn := range extraction.Funcs {
				if !strings.HasSuffix(strings.ToLower(fn.Name), ".leaf") {
					continue
				}
				seen = true
				if len(fn.ForceTypeAssertHits) != test.want {
					t.Fatalf("got %+v, want %d warnings", fn.ForceTypeAssertHits, test.want)
				}
			}
			if !seen {
				t.Fatal("missing helper function")
			}
		})
	}
}

func TestPrivateAssertionCacheInvalidatesCaller(t *testing.T) {
	root, _ := sourceFixModule(t, map[string]string{
		"leaf.go":   `package sample;func leaf(v any){_=v.(string)}`,
		"caller.go": `package sample;func caller(v any){if _,ok:=v.(string);ok{leaf(v)}}`,
	})
	cache := &FixExtractor{}
	before, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	assertPrivateWarnings(t, before.Funcs, 0)
	caller := filepath.Join(root, "caller.go")
	cache.Invalidate([]string{caller})
	overlay := SourceOverlay{Files: map[string][]byte{caller: []byte(`package sample;func caller(v any){leaf(v)}`)}, Disk: before.Sources.EvidenceOnly()}
	after, err := cache.Extract(context.Background(), root, []string{"."}, overlay)
	if err != nil {
		t.Fatal(err)
	}
	assertPrivateWarnings(t, after.Funcs, 1)
	full, err := ExtractFixOverlay(context.Background(), root, []string{"."}, overlay)
	if err != nil {
		t.Fatal(err)
	}
	assertPrivateWarnings(t, full.Funcs, 1)
}

func TestPrivateAssertionIncludesTestCallers(t *testing.T) {
	root, withTests := sourceFixModule(t, map[string]string{
		"leaf.go":      `package sample;func leaf(v any){_=v.(string)};func caller(){leaf("safe")}`,
		"leaf_test.go": `package sample;func testCaller(v any){leaf(v)}`,
	})
	assertPrivateWarnings(t, withTests.Funcs, 1)
	production, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	assertPrivateWarnings(t, production.Funcs, 0)
}

func assertPrivateWarnings(t *testing.T, functions []FuncPdg, want int) {
	t.Helper()
	for _, function := range functions {
		if strings.HasSuffix(function.Name, ".leaf") {
			if len(function.ForceTypeAssertHits) != want {
				t.Fatalf("got %+v, want %d warnings", function.ForceTypeAssertHits, want)
			}
			return
		}
	}
	t.Fatal("missing leaf function")
}

func TestPrivateAssertionRequiresCanonicalCalls(t *testing.T) {
	for _, missing := range []string{"caller flow", "caller operand", "caller identity", "helper operand", "foreign code"} {
		t.Run(missing, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;func leaf(v any){_=v.(string)};func caller(v any){if _,ok:=v.(string);ok{leaf(v)}}`)
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
			assertPrivateWarnings(t, analysis.result, 0)
			changed := false
			if missing == "foreign code" {
				pkg.OtherFiles = []string{"unknown.s"}
				changed = true
			}
			for _, source := range analysis.graphs {
				graph := source.graph
				if missing == "caller flow" && source.declaration.Name.Name == "caller" {
					graph.NormalFlow, graph.ValueFlow, graph.Extensions = nil, nil, nil
					changed = true
				}
				nodes, _ := assertionSources(source.declaration, pkg.Fset, graph)
				if missing == "caller identity" {
					for ref, node := range nodes {
						if call, ok := node.(*ast.CallExpr); ok {
							if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "leaf" {
								graph.Attributes[graph.Nodes[ref-1].Attributes-1].Callee = pdg.Fact{}
								changed = true
							}
						}
					}
				}
				if !strings.HasSuffix(missing, "operand") {
					continue
				}
				kept := graph.Edges[:0]
				for _, edge := range graph.Edges {
					node := nodes[edge.Target]
					_, call := node.(*ast.CallExpr)
					_, assertion := node.(*ast.TypeAssertExpr)
					selected := missing == "caller operand" && call || missing == "helper operand" && assertion && source.declaration.Name.Name == "leaf"
					if edge.Kind == pdg.Data && edge.Position == 1 && selected {
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
			assertPrivateWarnings(t, analysis.result, 1)
		})
	}
}
