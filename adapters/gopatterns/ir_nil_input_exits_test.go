package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestNilInputValuesComposeExitEffects(t *testing.T) {
	for _, tc := range []struct {
		name, body, helpers string
		want                callflow.Outcomes
	}{
		{"nil dereference", `var p *int;_=*p`, "", callflow.Panics},
		{"non nil dereference", `value:=0;p:=&value;_=*p`, "", callflow.Returns},
		{"nil field", `var p *S;_=p.Value`, `type S struct{Value int}`, callflow.Panics},
		{"non nil field", `value:=S{};p:=&value;_=p.Value`, `type S struct{Value int}`, callflow.Returns},
		{"nil bound value method", `var p *S;_=p.Method`, `type S struct{};func(S)Method(){}`, callflow.Panics},
		{"non nil bound value method", `value:=S{};p:=&value;_=p.Method`, `type S struct{};func(S)Method(){}`, callflow.Returns},
		{"nil bound pointer method", `var p *S;_=p.Method`, `type S struct{};func(*S)Method(){}`, callflow.Returns},
		{"nil interface method", `var value I;_=value.Method`, `type I interface{Method()}`, callflow.Panics},
		{"boxed nil interface method", `var p *S;var value I=p;_=value.Method`, `type I interface{Method()};type S struct{};func(*S)Method(){}`, callflow.Returns},
		{"nil argument", `read(nil)`, `func read(p *int){_=*p}`, callflow.Panics},
		{"non nil argument", `value:=0;read(&value)`, `func read(p *int){_=*p}`, callflow.Returns},
		{"nil return", `p:=pointer();_=*p`, `func pointer()*int{return nil}`, callflow.Panics},
		{"guarded unknown", `read(pointer)`, `var pointer *int;func read(p *int){if p!=nil{_=*p}}`, callflow.Returns},
		{"guarded nil", `read(pointer)`, `var pointer *int;func read(p *int){if p==nil{_=*p}}`, callflow.Returns | callflow.Panics},
		{"captured nil", `var p *int;func(){_=*p}()`, "", callflow.Panics},
		{"recover nil fault", `defer func(){recover()}();var p *int;_=*p`, "", callflow.Returns},
		{"panic in defer", `defer func(){var p *int;_=*p}()`, "", callflow.Panics},
		{"unknown dereference", `_=*pointer`, `var pointer *int`, callflow.Returns | callflow.Panics},
		{"nil promoted field", `var p *Outer;_=p.Value`, `type S struct{Value int};type Outer struct{*S}`, callflow.Returns | callflow.Panics},
		{"nested pointer remains unknown", `value:=S{};p:=&value;_=*p.Pointer`, `type S struct{Pointer *int}`, callflow.Returns | callflow.Panics},
		{"sibling fixes nil", `var p *int;_=*p+fix(&p)`, `func fix(p **int)int{value:=0;*p=&value;return 0}`, callflow.Returns | callflow.Panics},
		{"sibling clears pointer", `value:=0;p:=&value;_=*p+clearPointer(&p)`, `func clearPointer(p **int)int{*p=nil;return 0}`, callflow.Returns | callflow.Panics},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;func f(){`+tc.body+`};`+tc.helpers)
			graphs := map[string]*pdg.Graph{}
			for _, graph := range extraction.Callables {
				graphs[graph.Tables.Text(graph.Function.ID)] = graph
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries["example.f"].Ordinary; got != tc.want {
				t.Fatalf("exit=%v want %v", got, tc.want)
			}
		})
	}
}

func TestNilInputExitsPruneSharedGraph(t *testing.T) {
	for _, tc := range []struct {
		name, body, helpers string
		reachable           bool
	}{
		{"direct nil", `var p *int;_=*p;println("after")`, "", false},
		{"helper nil", `read(nil);println("after")`, `func read(p *int){_=*p}`, false},
		{"non nil", `value:=0;p:=&value;_=*p;println("after")`, "", true},
		{"mixed paths", `var p *int;if choose{value:=0;p=&value};_=*p;println("after")`, "", true},
		{"recovery does not resume", `defer func(){recover()}();var p *int;_=*p;println("after")`, "", false},
		{"sibling fixes nil", `var p *int;_=*p+fix(&p);println("after")`, `func fix(p **int)int{value:=0;*p=&value;return 0}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`};`+tc.helpers)
			graph := continuationGraph(t, extraction, "example.f")
			if err := pdg.Validate(graph); err != nil {
				t.Fatal(err)
			}
			found, reachable := false, false
			for _, block := range graph.ValueFlow.Blocks {
				for _, ref := range block.Operations {
					attrs := graph.NodeAttributes(graph.Nodes[ref-1])
					if graph.Tables.Text(attrs.Callee.Value) != "println" {
						continue
					}
					found = true
					reachable = reachable || block.Reachable
					if !block.Reachable {
						assertNoResultExecutionEdges(t, graph, ref)
					}
				}
			}
			if !found {
				t.Fatal("missing source operation")
			}
			if reachable != tc.reachable {
				t.Fatalf("after reachable=%v want %v", reachable, tc.reachable)
			}
			if !tc.reachable && graph.ValueFlow.ExitContext == nil {
				t.Fatal("missing original exit context")
			}
		})
	}
}
