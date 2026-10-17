package gopatterns

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"golang.org/x/tools/go/packages"
	"strings"
	"testing"
)

func TestNilErrPathBoundaries(t *testing.T) {
	cases := []struct {
		name, source string
		want         int
	}{
		{"copied known boolean", `func f(err error,b,c bool)error{if err!=nil{if c{b=c;if !b{return nil};return err};return err};return nil}`, 0},
		{"copied unknown boolean", `func f(err error,c bool)error{b:=c;if err!=nil{if b{return err};if !c{return err};return nil};return nil}`, 0},
		{"copy survives source overwrite", `func f(err error,b bool)error{a:=b;b=false;if err!=nil{if a{return err};if !a{return err};return nil};return nil}`, 0},
		{"negated copy", `func f(err error,b,c bool)error{if err!=nil{if c{b=!c;if b{return nil};return err};return err};return nil}`, 0},
		{"typed nil error return nonnil", `func f(err error)error{if err!=nil{var out *pointerError;return out};return nil}`, 0},
		{"typed nil assigned error nonnil", `func f(err error)error{if err!=nil{var p *pointerError;var out error=p;return out};return nil}`, 0},
		{"typed nil assigned named error nonnil", `func f(err error)(out error){if err!=nil{var p *pointerError;out=p;return};return nil}`, 0},
		{"typed nil any return nonnil", `func f(err error)(any,error){if err!=nil{var p *int;return p,nil};return nil,nil}`, 0},
		{"initializer replaces empty fallback", `func f(fetch func()(Data,error))(Data,error){result:=Data{};var err error;if result,err=fetch();err!=nil{return result,nil};return result,nil}`, 0},
		{"initializer replaces nil state propagated", `func f(fetch func()error)([]byte,error){var err error;var result []byte;if result,err=produce(fetch);err!=nil{return nil,err};return result,nil}`, 0},
		{"initializer replaces nil state lost", `func f(fetch func()error)([]byte,error){var err error;var result []byte;if result,err=produce(fetch);err!=nil{return nil,nil};return result,nil}`, 1},
		{"named error guard propagated", `func f(fetch func()error)(err error){if err=fetch();err!=nil{return};return nil}`, 0},
		{"known nil guard unreachable", `func f()error{var err error;if err!=nil{return nil};return nil}`, 0},
		{"folded numeric conversion zero", `func f(err error)(int,error){if err!=nil{return int(0),nil};return 0,err}`, 1},
		{"folded numeric conversion nonzero", `func f(err error)(int,error){if err!=nil{return int(1),nil};return 0,err}`, 0},
		{"nil conversion guard unreachable", `func f()error{err:=error(nil);if err!=nil{return nil};return nil}`, 0},
		{"blank declaration", `func f(err error)error{if err!=nil{var _ error;return nil};return nil}`, 1},
		{"blank constant", `func f(err error)error{if err!=nil{const _=0;return nil};return nil}`, 1},
		{"inverse guard fallthrough loses", `func f(err error)error{if err==nil{return nil};return nil}`, 1},
		{"inverse guard fallthrough propagates", `func f(err error)error{if err==nil{return nil};return err}`, 0},
		{"inverse guard fallback", `func f(err error,hostname string)(string,error){if err==nil{return hostname,nil};return hostname,nil}`, 0},
		{"generic zero error unknown", `func f[T error](err error)(result T){if err!=nil{return};return}`, 0},
		{"generic local error unknown", `func f[T error](err error)error{if err!=nil{var result T;return result};return nil}`, 0},
		{"boxed zero return nonnil", `func f(err error)(any,error){if err!=nil{return Data{},nil};return nil,nil}`, 0},
		{"boxed field nonzero", `func f(err error)(Payload,error){if err!=nil{return Payload{V:0},nil};return Payload{},nil}`, 0},
		{"boxed array nonzero", `func f(err error)([1]any,error){if err!=nil{return [1]any{0},nil};return [1]any{},nil}`, 0},
		{"nil interface field zero", `func f(err error)(Payload,error){if err!=nil{return Payload{V:nil},nil};return Payload{},nil}`, 1},
		{"unused closure defer", `func f(err error)error{_ = func(){defer func(){}()};if err!=nil{return nil};return nil}`, 1},
		{"error identity conversion", `func f(err error)error{if err!=nil{if error(err)==nil{return err};return nil};return nil}`, 1},
		{"known false outer branch", `func f(err error)error{if false{if err!=nil{return nil}};return err}`, 0},
		{"enclosing boolean", `func f(err error,b bool)error{if b{if err!=nil{if !b{return nil};return err}};return err}`, 0},
		{"predecessor return", `func f(err error,b bool)error{if b{return err};if err!=nil{if b{return nil};return err};return nil}`, 0},
		{"correlated branches", `func f(err error,b bool)error{if err!=nil{if b{return err};if !b{return err};return nil};return nil}`, 0},
		{"named nil error", `func f(err error)(result error){if err!=nil{return};return err}`, 1},
		{"local nil error", `func f(err error)error{if err!=nil{var result error;return result};return err}`, 1},
		{"concrete zero error nonnil", `func f(err error)error{if err!=nil{return valueError{}};return nil}`, 0},
		{"nonempty slice fallback", `func f(err error)([]int,error){if err!=nil{return []int{0},nil};return nil,nil}`, 0},
		{"nonempty map fallback", `func f(err error)(map[int]int,error){if err!=nil{return map[int]int{0:0},nil};return nil,nil}`, 0},
		{"aggregate fallback is nonzero", `func f(err error)(Data,error){if err!=nil{return Data{N:1},nil};return Data{},nil}`, 0},
		{"parallel initialization mutable binding", `func f(err error,b bool)error{a,c:=0,0;_=c;if b{a=1};if err!=nil{if a==1{return nil};return err};return nil}`, 1},
		{"constant comparison", `func f(err error)error{if err!=nil{n:=1;if n==1{return err};return nil};return nil}`, 0},
		{"short circuit false", `func f(err error,b bool)error{if err!=nil{if b&&false{return nil};return err};return nil}`, 0},
		{"promoted receiver overwrite", `func f(p *Promoted)error{if p.Err!=nil{p.R=&R{};return nil};return nil}`, 0},
		{"whole receiver overwrite", `func f(p *R)error{if p.Err!=nil{*p=R{};return nil};return nil}`, 0},
		{"alias before guard", `func f(p *R)error{alias:=p;if p.Err!=nil{alias.Err=nil;return nil};return nil}`, 0},
		{"pointer slot alias", `func f(p *R)error{alias:=&p.Err;if p.Err!=nil{*alias=nil;return nil};return nil}`, 0},
		{"opaque receiver callback", `func f(p *R,repair func(*R))error{if p.Err!=nil{repair(p);return nil};return nil}`, 0},
		{"known panic", `func f(err error)error{if err!=nil{panic("stop");return nil};return nil}`, 0},
		{"panic before guard", `func f(err error)error{panic("stop");if err!=nil{return nil};return nil}`, 0},
		{"index target may panic", `func f(err error)error{if err!=nil{s:=[]int{};s[0]=1;return nil};return nil}`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, "package p;"+tc.source+`;func produce(fetch func()error)([]byte,error){return nil,fetch()};type R struct{Err error};type Promoted struct{*R};type Data struct{N int};type Payload struct{V any};type pointerError struct{};func(*pointerError)Error()string{return "failure"};type valueError struct{};func(valueError)Error()string{return "failure"}`)
			if got := findNilErrHits(fn, fs, info); len(got) != tc.want {
				t.Fatalf("findings=%d,want %d: %+v", len(got), tc.want, got)
			}
		})
	}
}

func TestNilErrCompilerFieldIdentity(t *testing.T) {
	file, fs, info, _ := typeCheck(t, `package p;type R struct{Err,Other error};type Promoted struct{*R};func f(p,q *Promoted){_=p.Err;_=p.R.Err;_=q.Err;_=p.Other;_=p.R}`)
	fn := findFunc(file, "f")
	var values []ast.Expr
	for _, statement := range fn.Body.List {
		values = append(values, statement.(*ast.AssignStmt).Rhs[0])
	}
	graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	query := newNilErrGraph(fn, fs, &graph)
	query.indexTypedBindings(fn, info)
	if err := query.indexValueFacts(info); err != nil {
		t.Fatal(err)
	}
	paths := make([]bindingflow.Path, len(values))
	for ref, node := range query.source {
		for i, value := range values {
			if node == value {
				paths[i] = query.copies.Path(ref, query.valueFacts)
			}
		}
	}
	if paths[0].Root == 0 || !paths[0].Same(paths[1]) {
		t.Fatal("promoted path differs from explicit path")
	}
	for _, other := range paths[2:4] {
		if other.Root == 0 || paths[0].Same(other) {
			t.Fatal("different receiver or field equated")
		}
	}
	if !paths[4].PrefixOf(paths[0]) {
		t.Fatal("embedded receiver replacement not recognized")
	}

}

func TestNilErrPathBudgetIsQuiet(t *testing.T) {
	source := `package p;func f(err error)error{if err!=nil{` + strings.Repeat(`_=0;`, nilErrPathBudget+1) + `return nil};return nil}`
	fn, fs, info := parseNilErrFunc(t, source)
	if got := findNilErrHits(fn, fs, info); len(got) != 0 {
		t.Fatalf("budget exhaustion invented evidence: %+v", got)
	}
}

func TestNilErrKnownTerminalCalls(t *testing.T) {
	for _, call := range []string{`os.Exit(1)`, `runtime.Goexit()`, `log.Fatal("stop")`, `log.Panic("stop")`} {
		file, fs, info, _ := typeCheck(t, `package p;import("os";"runtime";"log");var _,_,_=os.Exit,runtime.Goexit,log.Fatal;func f(err error)error{`+call+`;if err!=nil{return nil};return nil}`)
		if got := findNilErrHits(findFunc(file, "f"), fs, info); len(got) != 0 {
			t.Fatalf("%s: unreachable finding %+v", call, got)
		}
	}
}
