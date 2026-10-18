package gopatterns

import (
	"errors"
	"fmt"
	"github.com/rstyled/gstyle/domain/outputflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"io"
	"testing"
)

func TestOutputSuccessEvidence(t *testing.T) {
	cases := []struct {
		name, body string
		report     bool
	}{
		{"success return", `fmt.Fprintln(w,err);return nil`, true},
		{"blank error", `_,_=fmt.Fprintln(w,err);return nil`, true},
		{"nil branch result", `if err!=nil{fmt.Fprintln(w,err);return nil};return nil`, true},

		{"original error", `if err!=nil{fmt.Fprintln(w,err);return err};return nil`, false},
		{"nested diagnostic", `if err!=nil{if w!=nil{fmt.Fprintln(w,err)};return err};return nil`, false},
		{"unknown return", `fmt.Fprintln(w,err);return err`, false},
		{"unknown call between write and return", `fmt.Fprintln(w,err);other();return nil`, false},
		{"nil is shadowed", `nil:=err;fmt.Fprintln(w,err);return nil`, false},
		{"nil variable", `var result error;fmt.Fprintln(w,err);return result`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `package fixture;import("fmt";"io");func other(){};func f(w io.Writer,err error)error{` + tc.body + `}`
			file, fset, info, pkg := typeCheck(t, src)
			sites, success := graphErrorCheckFacts(t, findFunc(file, "f"), fset, info, pkg)
			discarded := 0
			for _, site := range sites {
				if site.DiscardedOutput {
					discarded++
				}
			}
			if discarded != 1 {
				t.Fatalf("raw discarded calls=%d,want 1", discarded)
			}
			if (len(success) == 1) != tc.report {
				t.Fatalf("success sites=%v,want report=%v", success, tc.report)
			}
		})
	}
}

func TestOutputSuccessReturnShapes(t *testing.T) {
	cases := []struct {
		name, source string
		report       bool
	}{
		{"tuple", `func f(w io.Writer)(int,error){fmt.Fprintln(w,"x");return 0,nil}`, true},
		{"named result", `func f(w io.Writer)(err error){fmt.Fprintln(w,"x");return nil}`, true},
		{"deferred named replacement", `func f(w io.Writer)(err error){defer func(){err=fmt.Errorf("failed")}();fmt.Fprintln(w,"x");return nil}`, false},
		{"deferred unnamed panic", `func f(w io.Writer)error{defer func(){panic("stop")}();fmt.Fprintln(w,"x");return nil}`, false},
		{"normal defer unknown", `func f(w io.Writer)error{defer func(){}();fmt.Fprintln(w,"x");return nil}`, false},
		{"void best effort", `func f(w io.Writer){fmt.Fprintln(w,"x")}`, false},
		{"exit callback", `func f(w io.Writer,err error,exit func(int)){if err!=nil{fmt.Fprintln(w,err);exit(1)}}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, info, pkg := typeCheck(t, `package fixture;import("fmt";"io");`+tc.source)
			_, success := graphErrorCheckFacts(t, findFunc(file, "f"), fset, info, pkg)
			if (len(success) == 1) != tc.report {
				t.Fatalf("success sites=%v,want report=%v", success, tc.report)
			}
		})
	}
}

type failedDiagnosticWriter struct{ writes int }

func (w *failedDiagnosticWriter) Write([]byte) (int, error) { w.writes++; return 0, io.ErrClosedPipe }
func diagnosticThenFailure(w io.Writer, err error) error {
	if err != nil {
		fmt.Fprintln(w, err)
		return err
	}
	return nil
}
func outputThenSuccess(w io.Writer) error { fmt.Fprintln(w, "result"); return nil }

func outputArithmeticThenSuccess(w io.Writer, n int) error {
	fmt.Fprintln(w, n+1)
	return nil
}
func TestFailedWriteBehavior(t *testing.T) {
	cause := errors.New("read failed")
	diagnostic := &failedDiagnosticWriter{}
	if got := diagnosticThenFailure(diagnostic, cause); diagnostic.writes != 1 || !errors.Is(got, cause) {
		t.Fatalf("diagnostic writes=%d,result=%v", diagnostic.writes, got)
	}
	arithmetic := &failedDiagnosticWriter{}
	if got := outputArithmeticThenSuccess(arithmetic, 3); arithmetic.writes != 1 || got != nil {
		t.Fatalf("arithmetic writes=%d,result=%v", arithmetic.writes, got)
	}
	output := &failedDiagnosticWriter{}
	if got := outputThenSuccess(output); output.writes != 1 || got != nil {
		t.Fatalf("output writes=%d,result=%v", output.writes, got)
	}
}

func TestRuntimeTypeParameterIsNotAnInterfaceValue(t *testing.T) {
	file, _, info, _ := typeCheck(t, `package p;func f[T ~int](value T){}`)
	parameter := findFunc(file, "f").Type.TypeParams.List[0].Names[0]
	if got := runtimeTypeKind(info.Defs[parameter].Type()); got != "" {
		t.Fatalf("type parameter classified as runtime %q", got)
	}
}

func graphErrorCheckFacts(t *testing.T, fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, pkg *types.Package) ([]patterns.ErrorCheckSite, []int) {
	t.Helper()
	loaded := &packages.Package{Fset: fset, TypesInfo: info, Types: pkg, PkgPath: pkg.Path()}
	graph, err := buildGraph(loaded, fn, "test.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	sites, success, err := reachableErrorCheckFacts(fn, fset, info, &graph)
	if err != nil {
		t.Fatal(err)
	}
	return sites, success
}

func TestOutputSuccessUsesCanonicalContinuations(t *testing.T) {
	cases := []struct {
		name, body string
		want       bool
	}{
		{"all branches succeed", `fmt.Fprintln(w,err);if c{return nil};return nil`, true},
		{"one branch fails", `fmt.Fprintln(w,err);if c{return err};return nil`, false},
		{"type switch completes", `fmt.Fprintln(w,err);switch v:=a.(type){case int:_=v;return nil;default:return nil}`, true},
		{"goto return", `fmt.Fprintln(w,err);goto done;done:return nil`, true},
		{"possible infinite path", `fmt.Fprintln(w,err);for c{};return nil`, false},
		{"normal local work", `fmt.Fprintln(w,err);x:=n+1;_=x;return nil`, true},
		{"interface comparison may panic", `fmt.Fprintln(w,err);_=a==b;return nil`, false},
		{"interface switch may panic", `fmt.Fprintln(w,err);switch a{case b:return nil};return nil`, false},
		{"nil interface comparison is safe", `fmt.Fprintln(w,err);_=a==nil;return nil`, true},
		{"field address may panic", `fmt.Fprintln(w,err);var p *struct{value int};_=&p.value;return nil`, false},
		{"index address may panic", `fmt.Fprintln(w,err);var p *[1]int;_=&p[0];return nil`, false},
		{"division may panic", `fmt.Fprintln(w,err);_=1/n;return nil`, false},
		{"negative shift may panic", `fmt.Fprintln(w,err);_=1<<n;return nil`, false},
		{"unreachable write", `return nil;fmt.Fprintln(w,err);return nil`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := `package p;import("fmt";"io");func f(w io.Writer,err error,c bool,a,b any,n int)error{` + tc.body + `}`
			file, fset, info, pkg := typeCheck(t, source)
			_, sites := graphErrorCheckFacts(t, findFunc(file, "f"), fset, info, pkg)
			if (len(sites) == 1) != tc.want {
				t.Fatalf("success sites=%v, want %v", sites, tc.want)
			}
		})
	}
}

func TestOutputCompletionRejectsInvalidContext(t *testing.T) {
	source := `package p;func f(run func(),p *int)error{run();_=*p;return nil}`
	for _, slot := range []pdg.Position{0, 2} {
		graph := parseIR(t, source, pdg.NewBuilder())
		if _, err := outputflow.NewSuccessFlow(&graph, slot); err == nil {
			t.Fatal("invalid error-result slot accepted")
		}
	}
	graph := parseIR(t, source, pdg.NewBuilder())
	proof, err := outputflow.NewSuccessFlow(&graph, 1)
	if err != nil {
		t.Fatal(err)
	}
	if proof.After(0) || proof.After(pdg.Ref(len(graph.Nodes)+1)) {
		t.Fatal("invalid call site accepted")
	}
	graph.ValueFlow.Blocks[0].Successors = []pdg.Ref{pdg.Ref(len(graph.ValueFlow.Blocks) + 1)}
	if _, err := outputflow.NewSuccessFlow(&graph, 1); err == nil {
		t.Fatal("invalid operation-flow successor accepted")
	}
}
