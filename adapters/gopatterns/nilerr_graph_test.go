package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"testing"
)

func TestNilErrCanonicalTransfers(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"goto guard", `goto check;check:if err!=nil{return nil};return err`, 1},
		{"goto skips loss", `if err!=nil{goto handled;return nil};handled:return err`, 0},
		{"goto reaches loss", `if err!=nil{goto lost};return err;lost:return nil`, 1},
		{"break reaches loss", `if err!=nil{for{break};return nil};return err`, 1},
		{"infinite path", `if err!=nil{for{}};return err`, 0},
		{"unreachable initializer", `var result int;goto check;result=1;check:if err!=nil{return result,nil};return 0,err`, 1},
		{"increment invalidates zero", `result:=0;result++;if err!=nil{return result,nil};return 0,err`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := "error"
			if tc.name == "unreachable initializer" || tc.name == "increment invalidates zero" {
				result = "(int,error)"
			}
			fn, fs, info := parseNilErrFunc(t, `package p;func f(err error)`+result+`{`+tc.body+`}`)
			if got := findNilErrHits(fn, fs, info); len(got) != tc.want {
				t.Fatalf("findings=%v want %d", got, tc.want)
			}
		})
	}
}
func TestNilErrUsesGraphSuccessors(t *testing.T) {
	fn, fs, info := parseNilErrFunc(t, `package p;func f(err error)error{if err!=nil{return nil};return err}`)
	graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	if hits := findNilErrGraphHits(fn, fs, info, &graph); len(hits) != 1 {
		t.Fatalf("original path: %v", hits)
	}
	query := newNilErrGraph(fn, fs, &graph)
	changed := false
	for i, block := range graph.ValueFlow.Blocks {
		if query.guards[block.Condition] == nil {
			continue
		}
		// Change the error branch to propagation while preserving two outcomes.
		graph.ValueFlow.Blocks[i].Successors[0] = block.Successors[1]
		changed = true
	}
	if !changed {
		t.Fatal("missing canonical guard")
	}
	graph.ValueFlow.RefreshReachability()
	if hits := findNilErrGraphHits(fn, fs, info, &graph); len(hits) != 0 {
		t.Fatalf("AST path overrides graph: %v", hits)
	}
}
func TestNilErrDoesNotUseLegacyPartialFlow(t *testing.T) {
	fn, fs, info := parseNilErrFunc(t, `package p;func f(err error)error{if err!=nil{return nil};return err}`)
	graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	graph.Extensions = nil
	graph.NormalFlow = nil
	if hits := findNilErrGraphHits(fn, fs, info, &graph); len(hits) != 0 {
		t.Fatalf("legacy flow certifies a finding: %v", hits)
	}
}

// Existing source regressions exercise the production graph path.
func findNilErrHits(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) []patterns.NilErrHit {
	if fn.Body == nil || len(errorResultPositions(fn, info)) == 0 {
		return nil
	}
	pkg := &packages.Package{Fset: fset, TypesInfo: info}
	if object := info.Defs[fn.Name]; object != nil && object.Pkg() != nil {
		pkg.Types = object.Pkg()
		pkg.PkgPath = object.Pkg().Path()
	}
	graph, err := buildGraph(pkg, fn, fset.PositionFor(fn.Pos(), false).Filename, pdg.NewBuilder())
	if err != nil {
		panic(err) // A broken test graph must fail the test, not look like no finding.
	}
	return findNilErrGraphHits(fn, fset, info, &graph)
}

func TestNilErrDoesNotInventRelationalPaths(t *testing.T) {
	fn, fs, info := parseNilErrFunc(t, `package p;func f(err error,n int)error{if n>0{if n<=0{if err!=nil{return nil}}};return err}`)
	if hits := findNilErrHits(fn, fs, info); len(hits) != 0 {
		t.Fatalf("infeasible relation produces a finding: %v", hits)
	}
}
