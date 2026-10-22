package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"path/filepath"
	"strings"
	"testing"
)

func TestNullableReturnAssertions(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example
 type Target struct{}
 type Other struct{}
 func factory(flag bool)any{if flag{return nil};return &Target{}}
 func mixed(flag bool)any{if flag{return &Other{}};return &Target{}}
 func unknown(v any)any{return v}
 func allNil()any{return nil}
 func deferred(flag bool)any{defer func(){}();if flag{return nil};return &Target{}}
 func recovered(flag bool)any{defer func(){recover()}();if flag{panic("bad")};return &Target{}}
 func named()(v any){defer func(){v=&Other{}}();return &Target{}}
 func typedNil(flag bool)any{if flag{return nil};return (*Target)(nil)}
 func nullableError(flag bool)(any,error){if flag{return nil,nil};return &Target{},nil}
 func replace(v *any){*v=&Other{}}
 var global any
 func checkGuard(flag bool){v:=factory(flag);if v==nil{return};_=v.(*Target)}
 func checkBranch(flag bool){v:=factory(flag);if v!=nil{_=v.(*Target)}}
 func checkElse(flag bool){v:=factory(flag);if v==nil{}else{_=v.(*Target)}}
 func checkReverse(flag bool){v:=factory(flag);if nil==v{return};_=v.(*Target)}
 func checkNegated(flag bool){v:=factory(flag);if !(v!=nil){return};_=v.(*Target)}
 func checkAlias(flag bool){v:=factory(flag);copy:=v;if copy==nil{return};_=copy.(*Target)}
 func checkDeferred(flag bool){v:=deferred(flag);if v==nil{return};_=v.(*Target)}
 func checkRecovered(flag bool){v:=recovered(flag);if v==nil{return};_=v.(*Target)}
 func checkTypedNil(flag bool){v:=typedNil(flag);if v==nil{return};_=v.(*Target)}
 func checkAny(flag bool){v:=factory(flag);if v==nil{return};_=v.(any)}
 func checkJoin(flag bool){var v any;if flag{v=factory(flag)}else{v=deferred(flag)};if v==nil{return};_=v.(*Target)}
 func checkUnguarded(flag bool){v:=factory(flag);_=v.(*Target)}
 func checkDirect(flag bool){_=factory(flag).(*Target)}
 func checkWrongBranch(flag bool){v:=factory(flag);if v==nil{_=v.(*Target)}}
 func checkWrongType(flag bool){v:=factory(flag);if v==nil{return};_=v.(*Other)}
 func checkMixed(flag bool){v:=mixed(flag);if v==nil{return};_=v.(*Target)}
 func checkUnknown(input any){v:=unknown(input);if v==nil{return};_=v.(*Target)}
 func checkAllNil(){v:=allNil();if v==nil{return};_=v.(*Target)}
 func checkOverwrite(flag bool){v:=factory(flag);if v==nil{return};v=&Other{};_=v.(*Target)}
 func checkOverwriteNil(flag bool){v:=factory(flag);if v==nil{return};v=nil;_=v.(*Target)}
 func checkBranchOverwrite(flag bool){v:=factory(flag);if v==nil{return};if flag{v=nil};_=v.(*Target)}
 func checkStaleGuard(flag bool){var v any=&Target{};if v==nil{return};v=factory(flag);_=v.(*Target)}
 func checkDifferentGuard(flag bool){v:=factory(flag);other:=factory(flag);if other==nil{return};_=v.(*Target)}
 func checkShadow(flag bool){v:=factory(flag);if v==nil{return};{v:=factory(flag);_=v.(*Target)}}
 func checkBypass(flag bool){v:=factory(flag);if flag{goto use};if v==nil{return};use:_=v.(*Target)}
 func checkPartialGuard(flag bool){v:=factory(flag);if flag{if v==nil{return}};_=v.(*Target)}
 func checkLoop(flag bool){v:=factory(flag);if v==nil{return};for flag{_=v.(*Target);v=factory(flag)}}
 func checkEscape(flag bool){v:=factory(flag);if v==nil{return};replace(&v);_=v.(*Target)}
 func checkCapture(flag bool){v:=factory(flag);if v==nil{return};func(){v=nil}();_=v.(*Target)}
 func checkGlobal(flag bool){global=factory(flag);if global==nil{return};_=global.(*Target)}
 func checkField(flag bool){box:=struct{v any}{factory(flag)};if box.v==nil{return};_=box.v.(*Target)}
 func checkNamed(){v:=named();if v==nil{return};_=v.(*Target)}
 func checkTypedComparison(flag bool){v:=factory(flag);if v==(*Target)(nil){return};_=v.(*Target)}
 func checkRecoveryUnguarded(flag bool){v:=recovered(flag);_=v.(*Target)}
 func checkErrorOnly(flag bool){v,err:=nullableError(flag);if err!=nil{return};_=v.(*Target)}
 `)
	safe := map[string]bool{"checkGuard": true, "checkBranch": true, "checkElse": true, "checkReverse": true, "checkNegated": true, "checkAlias": true, "checkDeferred": true, "checkRecovered": true, "checkTypedNil": true, "checkAny": true, "checkJoin": true}
	seen := 0
	for _, fn := range extraction.Funcs {
		name := fn.Name[strings.LastIndex(fn.Name, ".")+1:]
		if !strings.HasPrefix(name, "check") {
			continue
		}
		seen++
		want := 1
		if safe[name] {
			want = 0
		}
		if len(fn.ForceTypeAssertHits) != want {
			t.Errorf("%s: got %+v, want %d warnings", name, fn.ForceTypeAssertHits, want)
		}
	}
	if seen != 35 {
		t.Fatalf("checked %d functions, want 35", seen)
	}
}

func TestNullableReturnRequiresCanonicalGuard(t *testing.T) {
	for _, missing := range []string{"predicate operand", "predicate condition", "helper flow", "assertion type"} {
		t.Run(missing, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;type Target struct{};func factory(flag bool)any{defer func(){}();if flag{return nil};return &Target{}};func check(flag bool){v:=factory(flag);if v==nil{return};_=v.(*Target)}`)
			root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
			analysis := packageAnalysis{pkg: pkg, root: root, pool: pdg.NewBuilder()}
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
				if missing == "helper flow" && source.declaration.Name.Name == "factory" {
					graph.NormalFlow, graph.ValueFlow, graph.Extensions = nil, nil, nil
					changed = true
				}
				if source.declaration.Name.Name != "check" {
					continue
				}
				if missing == "assertion type" {
					ast.Inspect(source.declaration.Body, func(node ast.Node) bool {
						if assertion, ok := node.(*ast.TypeAssertExpr); ok {
							delete(pkg.TypesInfo.Types, assertion.Type)
							changed = true
						}
						return true
					})
				}
				if missing == "predicate condition" {
					flow := pdg.BodyFlow(graph)
					for i, block := range flow.Blocks {
						if block.Condition != 0 {
							flow.Blocks[i].Condition = 0
							changed = true
						}
					}
				}
				if missing == "predicate operand" {
					kept := graph.Edges[:0]
					for _, edge := range graph.Edges {
						target := graph.Nodes[edge.Target-1]
						if edge.Kind == pdg.Data && graph.Tables.Text(graph.NodeAttributes(target).Operator.Value) == "==" {
							changed = true
							continue
						}
						kept = append(kept, edge)
					}
					graph.Edges = kept
				}
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

func assertNullableWarnings(t *testing.T, functions []FuncPdg, want int) {
	t.Helper()
	for _, fn := range functions {
		if strings.HasSuffix(fn.Name, ".check") {
			if len(fn.ForceTypeAssertHits) != want {
				t.Fatalf("got %+v, want %d warnings", fn.ForceTypeAssertHits, want)
			}
			return
		}
	}
	t.Fatal("missing check function")
}

func TestNullableReturnCacheInvalidation(t *testing.T) {
	root, _ := sourceFixModule(t, map[string]string{
		"helper.go": `package sample;type Target struct{};type Other struct{};func factory(flag bool)any{if flag{return nil};return &Target{}}`,
		"caller.go": `package sample;func check(flag bool){v:=factory(flag);if v==nil{return};_=v.(*Target)}`,
	})
	cache := &FixExtractor{}
	before, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	assertNullableWarnings(t, before.Funcs, 0)
	helper := filepath.Join(root, "helper.go")
	cache.Invalidate([]string{helper})
	revision := SourceOverlay{Files: map[string][]byte{helper: []byte(`package sample;type Target struct{};type Other struct{};func factory(flag bool)any{if flag{return nil};return &Other{}}`)}, Disk: before.Sources.EvidenceOnly()}
	next, err := cache.Extract(context.Background(), root, []string{"."}, revision)
	if err != nil {
		t.Fatal(err)
	}
	full, err := ExtractFixOverlay(context.Background(), root, []string{"."}, revision)
	if err != nil {
		t.Fatal(err)
	}
	assertNullableWarnings(t, next.Funcs, 1)
	assertNullableWarnings(t, full.Funcs, 1)
}
