package gopatterns

import (
	"go/ast"
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestExitArgumentsSpecializeCanonicalCalls(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         callflow.Outcomes
	}{
		{"false argument", `func f(){stop(false)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns},
		{"true argument", `func f(){stop(true)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Panics},
		{"binding copy", `func f(){flag:=false;copy:=flag;stop(copy)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns},
		{"caller overwrite", `func f(){flag:=true;flag=false;stop(flag)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns},
		{"caller indirect overwrite stays conservative", `func f(){flag:=false;pointer:=&flag;*pointer=true;stop(flag)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns | callflow.Panics},
		{"callee indirect overwrite stays conservative", `func f(){stop(false)};func stop(flag bool){pointer:=&flag;*pointer=true;if flag{panic("stop")}}`, callflow.Returns | callflow.Panics},
		{"nested argument forwarding", `func f(){middle(false)};func middle(flag bool){stop(flag)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns},
		{"indirect recover keeps its frame", `func f(){defer middle(true);panic("initial")};func middle(flag bool){stop(flag)};func stop(flag bool){if flag{recover()}}`, callflow.Panics},
		{"callee overwrite", `func f(){stop(true)};func stop(flag bool){flag=false;if flag{panic("stop")}}`, callflow.Returns},
		{"nil pointer", `func f(){var pointer *int;stop(pointer)};func stop(pointer *int){if pointer!=nil{panic("stop")}}`, callflow.Returns},
		{"boxed nil pointer", `func f(){var pointer *int;stop(pointer)};func stop(value any){if value==nil{panic("stop")}}`, callflow.Returns},
		{"nil interface", `func f(){var value any;stop(value)};func stop(value any){if value==nil{panic("stop")}}`, callflow.Panics},
		{"two parameter positions", `func f(){stop(true,false)};func stop(first,second bool){if first&&!second{return};panic("stop")}`, callflow.Returns},
		{"finite recursive state", `func f(){walk(true)};func walk(flag bool){if !flag{return};walk(false)}`, callflow.Returns},
		{"infinite recursive state", `func f(){walk(true)};func walk(flag bool){if flag{walk(true)}}`, callflow.Diverges},
		{"unknown argument", `func f(flag bool){stop(flag)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns | callflow.Panics},
		{"stored declaration", `func f(){target:=stop;target(false)};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns},
		{"saved false defer", `func f(){flag:=false;defer stop(flag);flag=true;_=flag};func stop(flag bool){if flag{panic("stop")}}`, callflow.Returns},
		{"saved true defer", `func f(){flag:=true;defer stop(flag);flag=false;_=flag};func stop(flag bool){if flag{panic("stop")}}`, callflow.Panics},
		{"deferred recovery argument", `func f(){defer stop(true);panic("initial")};func stop(flag bool){if flag{recover()}}`, callflow.Returns},
		{"deferred ineligible recovery", `func f(){defer stop(false);panic("initial")};func stop(flag bool){if flag{recover()}}`, callflow.Panics},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graphs, name := argumentExitGraphs(t, tc.source)
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries[name].Ordinary; got != tc.want {
				t.Fatalf("exit=%v want %v", got, tc.want)
			}
		})
	}
}
func argumentExitGraphs(t *testing.T, source string) (map[string]*pdg.Graph, string) {
	t.Helper()
	pkg := loadTestPackage(t, "package p;"+source)
	graphs := map[string]*pdg.Graph{}
	for _, decl := range pkg.Syntax[0].Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
		if err != nil {
			t.Fatal(err)
		}
		graphs[graph.Tables.Text(graph.Function.ID)] = &graph
	}
	return graphs, pkg.PkgPath + ".f"
}

func nativeArgumentRecursion(flag bool, count *int) {
	*count++
	if flag {
		nativeArgumentRecursion(false, count)
	}
}
func TestNativeArgumentStateEndsRecursionAndSavesDeferInputs(t *testing.T) {
	count := 0
	nativeArgumentRecursion(true, &count)
	if count != 2 {
		t.Fatalf("calls=%d", count)
	}
	var saved []bool
	func() {
		flag := false
		defer func(value bool) { saved = append(saved, value) }(flag)
		flag = true
		defer func(value bool) { saved = append(saved, value) }(flag)
	}()
	if len(saved) != 2 || !saved[0] || saved[1] {
		t.Fatalf("saved=%v", saved)
	}
}
