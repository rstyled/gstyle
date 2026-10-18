package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestCallResultFacts(t *testing.T) {
	for _, tc := range []struct {
		name, body, helpers string
		want                callflow.Outcomes
	}{
		{"false", `if value(){panic("bad")}`, `func value()bool{return false}`, callflow.Returns},
		{"true", `if value(){panic("yes")}`, `func value()bool{return true}`, callflow.Panics},
		{"copy", `flag:=value();if flag{panic("bad")}`, `func value()bool{return false}`, callflow.Returns},
		{"argument context", `if value(false){panic("bad")}`, `func value(flag bool)bool{return flag}`, callflow.Returns},
		{"forwarding", `if value(false){panic("bad")}`, `func value(flag bool)bool{return child(flag)};func child(flag bool)bool{return flag}`, callflow.Returns},
		{"finite recursion", `if value(true){panic("bad")}`, `func value(flag bool)bool{if !flag{return false};return value(false)}`, callflow.Returns},
		{"pure recursion", `_=value()`, `func value()bool{return value()}`, callflow.Diverges},
		{"recursive finite exit", `if value(){panic("bad")}`, `func value()bool{if choose{return false};return value()}`, callflow.Returns | callflow.Diverges},
		{"mixed return", `if value(choose){panic("yes")}`, `func value(flag bool)bool{return flag}`, callflow.Returns | callflow.Panics},
		{"nil result", `if value()!=nil{panic("bad")}`, `func value()*int{return nil}`, callflow.Returns},
		{"boxed nil result", `if value()==nil{panic("bad")}`, `func value()any{var pointer *int;return pointer}`, callflow.Returns},
		{"tuple copy", `left,right:=value();if left||!right{panic("bad")}`, `func value()(bool,bool){return false,true}`, callflow.Returns},
		{"tuple forwarding", `left,right:=value();if left||!right{panic("bad")}`, `func value()(bool,bool){return child()};func child()(bool,bool){return false,true}`, callflow.Returns},
		{"tuple argument", `sink(value())`, `func value()(bool,bool){return false,true};func sink(a,b bool){if a||!b{panic("bad")}}`, callflow.Returns},
		{"named zero", `if value(){panic("bad")}`, `func value()(flag bool){return}`, callflow.Returns},
		{"named write", `if value(){panic("yes")}`, `func value()(flag bool){flag=true;return}`, callflow.Panics},
		{"unnamed closure defer remains unknown", `if !value(){panic("bad")}`, `func value()bool{flag:=true;defer func(){flag=false}();return flag}`, callflow.UnknownExit},
		{"named closure defer remains unknown", `if value(){panic("yes")}`, `func value()(flag bool){defer func(){flag=true}();return false}`, callflow.UnknownExit},
		{"unnamed snapshot with defer", `if !value(){panic("bad")}`, `func value()bool{defer noop();return true};func noop(){}`, callflow.Returns},
		{"named result with defer stays unknown", `if value(){panic("maybe")}`, `func value()(flag bool){defer noop();return false};func noop(){}`, callflow.Returns | callflow.Panics},
		{"unknown prefix", `if value(){panic("maybe")}`, `var foreign func();func value()bool{foreign();return false}`, callflow.UnknownExit},
		{"recovered panic stays unknown", `if value(){panic("maybe")}`, `func value()bool{defer absorb();panic("caught")};func absorb(){recover()}`, callflow.Returns | callflow.Panics},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`};`+tc.helpers)
			graphs := map[string]*pdg.Graph{}
			for i := range extraction.Funcs {
				graph := &extraction.Funcs[i].Pdg
				graphs[graph.Tables.Text(graph.Function.ID)] = graph
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries["example.f"].Ordinary; got != tc.want {
				t.Fatalf("exit outcomes=%v want %v", got, tc.want)
			}
		})
	}
}
