package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func callableValueCases() []unwindValueCase {
	return []unwindValueCase{
		{"callback false", `if apply(no){panic("bad")}`, `func no()bool{return false};func apply(cb func()bool)bool{return cb()}`, callflow.Returns},
		{"callback true", `if !apply(yes){panic("bad")}`, `func yes()bool{return true};func apply(cb func()bool)bool{return cb()}`, callflow.Returns},
		{"callback panic", `apply(stop);panic("unreachable")`, `func stop(){panic("yes")};func apply(cb func()){cb()}`, callflow.Panics},
		{"callback argument", `if apply(identity,false){panic("bad")}`, `func identity(v bool)bool{return v};func apply(cb func(bool)bool,v bool)bool{return cb(v)}`, callflow.Returns},
		{"forwarded callback", `if forward(no){panic("bad")}`, `func no()bool{return false};func apply(cb func()bool)bool{return cb()};func forward(cb func()bool)bool{return apply(cb)}`, callflow.Returns},
		{"recursive callback", `if apply(no,true){panic("bad")}`, `func no()bool{return false};func apply(cb func()bool,again bool)bool{if again{return apply(cb,false)};return cb()}`, callflow.Returns},
		{"recursive callback divergence", `apply(loop)`, `func loop(){loop()};func apply(cb func()){cb()}`, callflow.Diverges},
		{"callback reassigned", `if apply(yes,no){panic("bad")}`, `func no()bool{return false};func yes()bool{return true};func apply(cb,other func()bool)bool{cb=other;return cb()}`, callflow.Returns},
		{"returned declaration", `read:=value();if read(){panic("bad")}`, `func no()bool{return false};func value()func()bool{return no}`, callflow.Returns},
		{"returned identity callback", `read:=value(no);if read(){panic("bad")}`, `func no()bool{return false};func value(cb func()bool)func()bool{return cb}`, callflow.Returns},
		{"tuple callbacks", `a,b:=value();if a()||!b(){panic("bad")}`, `func no()bool{return false};func yes()bool{return true};func value()(func()bool,func()bool){return no,yes}`, callflow.Returns},
		{"joined targets", `read:=value(choose);if read(){panic("bad")}`, `func no()bool{return false};func alsoNo()bool{return false};func value(v bool)func()bool{if v{return no};return alsoNo}`, callflow.Returns},
		{"joined disagreement", `read:=value(choose);if read(){panic("possible")}`, `func no()bool{return false};func yes()bool{return true};func value(v bool)func()bool{if v{return no};return yes}`, callflow.Returns | callflow.Panics},
		{"capture free callback", `if apply(func()bool{return false}){panic("bad")}`, `func apply(cb func()bool)bool{return cb()}`, callflow.Returns},
		{"captured callback retains environment", `flag:=false;if apply(func()bool{return flag}){panic("possible")}`, `func apply(cb func()bool)bool{return cb()}`, callflow.Returns},
		{"returned captured environment is live", `read:=value(false);if read(){panic("possible")}`, `func value(flag bool)func()bool{return func()bool{return flag}}`, callflow.Returns},
		{"nil callback", `apply(nil)`, `func apply(cb func()){cb()}`, callflow.Panics},
		{"nil returned callback", `read:=value();read()`, `func value()func(){return nil}`, callflow.Panics},
		{"callback preserves captured cell", `flag:=false;keep:=func(){if flag{panic("bad")}};apply(no);keep()`, `func no()bool{return false};func apply(cb func()bool)bool{return cb()}`, callflow.Returns},
		{"callback indirect writer invalidates cell", `flag:=true;keep:=func(){if flag{panic("possible")}};apply(change,&flag);keep()`, `func change(p *bool){*p=false};func apply(cb func(*bool),p *bool){cb(p)}`, callflow.Returns | callflow.Panics},
		{"forwarded writer invalidates cell", `flag:=true;keep:=func(){if flag{panic("possible")}};forward(change,&flag);keep()`, `func change(p *bool){*p=false};func apply(cb func(*bool),p *bool){cb(p)};func forward(cb func(*bool),p *bool){apply(cb,p)}`, callflow.Returns | callflow.Panics},
		{"deferred writer invalidates cell", `flag:=true;keep:=func(){if flag{panic("possible")}};apply(change,&flag);keep()`, `func change(p *bool){*p=false};func apply(cb func(*bool),p *bool){defer cb(p)}`, callflow.Returns | callflow.Panics},
		{"nil callback preserves cells before recovery", `flag:=false;defer func(){if flag{panic("bad")};recover()}();apply(nil)`, `func apply(cb func()){cb()}`, callflow.Returns},

		{"captured callback writer reaches caller", `flag:=true;keep:=func(){if flag{panic("possible")}};apply(func(){flag=false});keep()`, `func apply(cb func()){cb()}`, callflow.Returns},
		{"deferred named callable result", `read:=value(yes,no);if read(){panic("bad")}`, `func no()bool{return false};func yes()bool{return true};func value(cb,other func()bool)(read func()bool){defer func(){read=other}();return cb}`, callflow.Returns},
		{"deferred unnamed callable snapshot", `read:=value();if read(){panic("bad")}`, `func no()bool{return false};func yes()bool{return true};func value()func()bool{read:=no;defer func(){read=yes}();return read}`, callflow.Returns},

		{"named function type conversion", `type Local func()bool;read:=Local(no);if apply(read){panic("bad")}`, `func no()bool{return false};func apply(cb func()bool)bool{return cb()}`, callflow.Returns},
		{"live captured callable", `read:=no;invoke:=func()bool{return read()};read=yes;if !invoke(){panic("bad")}`, `func no()bool{return false};func yes()bool{return true}`, callflow.Returns},

		{"callback panic prefix invalidates cell", `flag:=true;defer func(){if flag{panic("bad")};recover()}();apply(change,&flag)`, `func change(p *bool){*p=false;panic("caught")};func apply(cb func(*bool),p *bool){cb(p)}`, callflow.Returns | callflow.Panics},

		{"direct deferred callback recovers", `apply(catcher)`, `func catcher(){recover()};func apply(cb func()){defer cb();panic("caught")}`, callflow.Returns},
		{"nested deferred callback does not recover ancestor", `apply(catcher)`, `func catcher(){recover()};func apply(cb func()){defer func(){cb()}();panic("outer")}`, callflow.Panics},
		{"canonical Goexit callback", `apply(runtime.Goexit)`, `func apply(cb func()){cb()}`, callflow.ExitsGoroutine},

		{"deferred callback saves target", `if !value(yes,no){panic("bad")}`, `func no()bool{return false};func yes()bool{return true};func value(cb,other func()bool)(flag bool){defer func(saved func()bool){flag=saved()}(cb);cb=other;return false}`, callflow.Returns},
	}
}

func TestCallableValuesCrossInvocationBoundaries(t *testing.T) {
	cases := callableValueCases()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;import "runtime";var _=runtime.Goexit;var choose bool;func f(){`+tc.body+`};`+tc.helper)
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

func TestNativeCallableValueEvidence(t *testing.T) {
	var finite []unwindValueCase
	for _, tc := range callableValueCases() {
		if tc.want != callflow.Diverges {
			finite = append(finite, tc)
		}
	}
	verifyNativeValueCases(t, finite)
}
