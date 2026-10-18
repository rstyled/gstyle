package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"testing"
)

func TestExitFixedPointsUseActualRecoveryContexts(t *testing.T) {
	for _, tc := range []struct {
		name, source        string
		ordinary, inherited callflow.Outcomes
	}{
		{"guard removes ordinary cycle", `func f(){if recover()==nil{return};f()}`, callflow.Returns, callflow.RecoversInherited},
		{"guard removes deferred cycle", `func f(){if recover()!=nil{return};f()}`, callflow.Diverges, callflow.RecoversInherited},
		{"deferred context reaches aborting child", `func f(){defer f();if recover()==nil{os.Exit(1)}}`, callflow.Aborts, callflow.Aborts},
		{"unknown call remains unknown", `func f(){unknown()};func unknown()`, callflow.UnknownExit, callflow.UnknownExit},
		{"abort never invokes recursive defer", `func f(){defer f();os.Exit(1)}`, callflow.Aborts, callflow.Aborts},
		{"ordinary child breaks static cycle", `func f(){g()};func g(){if recover()==nil{return};f()}`, callflow.Returns, callflow.Returns},
		{"ordinary child retains true cycle", `func f(){g()};func g(){if recover()!=nil{return};f()}`, callflow.Diverges, callflow.Diverges},
		{"finite prefix reaches recursive continuation", `func f(){normal();f()};func normal(){}`, callflow.Diverges, callflow.Diverges},
		{"opaque prefix keeps uncertainty", `func f(){unknown();f()};func unknown()`, callflow.UnknownExit, callflow.UnknownExit},
		{"direct recursion", `func f(){f()}`, callflow.Diverges, callflow.Diverges},
		{"deferred recursion", `func f(){defer f()}`, callflow.Diverges, callflow.Diverges},
		{"panic through recursive defers", `func f(){defer f();panic("fault")}`, callflow.Diverges, callflow.Diverges},
		{"local loop", `func f(){for{}}`, callflow.Diverges, callflow.Diverges},
		{"contextual local loop", `func f(){if recover()==nil{return};for{}}`, callflow.Returns, callflow.Diverges},
		{"finite and infinite alternatives", `func f(){if choose{return};f()}`, callflow.Returns | callflow.Diverges, callflow.Returns | callflow.Diverges},
		{"panic exit", `func f(){panic("fault")}`, callflow.Panics, callflow.Panics},
		{"goroutine exit", `func f(){runtime.Goexit()}`, callflow.ExitsGoroutine, callflow.ExitsGoroutine},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import("os";"runtime");var _=os.Exit;var _=runtime.Goexit;var choose bool;`+tc.source)
			graphs := map[string]*pdg.Graph{}
			for _, declaration := range pkg.Syntax[0].Decls {
				fn, ok := declaration.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
				if err != nil {
					t.Fatal(err)
				}
				graphs[graph.Tables.Text(graph.Function.ID)] = &graph
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			got := summaries[pkg.PkgPath+".f"]
			if got.Ordinary != tc.ordinary || got.DeferredPanic != tc.inherited {
				t.Fatalf("ordinary=%v inherited=%v want %v/%v", got.Ordinary, got.DeferredPanic, tc.ordinary, tc.inherited)
			}
		})
	}
}

func nativeFiniteRecoveryRecursion(count *int) {
	*count++
	if recover() == nil {
		return
	}
	nativeFiniteRecoveryRecursion(count)
}
func TestNativeRecoveryBreaksStaticRecursiveCycle(t *testing.T) {
	ordinary := 0
	nativeFiniteRecoveryRecursion(&ordinary)
	if ordinary != 1 {
		t.Fatalf("ordinary calls=%d", ordinary)
	}
	inherited := 0
	func() { defer nativeFiniteRecoveryRecursion(&inherited); panic("initial") }()
	if inherited != 2 {
		t.Fatalf("inherited calls=%d", inherited)
	}
}
