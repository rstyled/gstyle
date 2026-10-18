package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
	"testing"
)

type unwindValueCase struct {
	name, body, helper string
	want               callflow.Outcomes
}

func unwindValueCases() []unwindValueCase {
	return []unwindValueCase{
		{"late true capture", `flag:=false;defer func(){if flag{panic("certain")}}();flag=true`, "", callflow.Panics},
		{"late false capture", `flag:=true;defer func(){if flag{panic("bad")}}();flag=false`, "", callflow.Returns},
		{"saved false argument", `flag:=false;defer func(saved bool){if saved&&flag{panic("bad")}}(flag);flag=true`, "", callflow.Returns},
		{"saved true argument live false", `flag:=true;defer func(saved bool){if saved&&flag{panic("bad")}}(flag);flag=false`, "", callflow.Returns},
		{"tuple arguments", `flag:=true;defer func(a,b bool){if a||!b||flag{panic("bad")}}(pair());flag=false`, `func pair()(bool,bool){return false,true}`, callflow.Returns},
		{"LIFO store before reader", `flag:=false;defer func(){if !flag{panic("bad")}}();defer func(){flag=true}()`, "", callflow.Returns},
		{"LIFO reader before store", `flag:=false;defer func(){flag=true}();defer func(){if flag{panic("bad")}}()`, "", callflow.Returns},
		{"ordinary panic postcondition", `flag:=false;defer func(){if !flag{panic("lost store")};recover()}();func(){flag=true;panic("caught")}()`, "", callflow.Returns},
		{"deferred panic postcondition", `flag:=false;defer func(){if !flag{panic("lost store")};recover()}();defer func(){flag=true;panic("caught")}()`, "", callflow.Returns},
		{"nested recovery preserves ancestor panic", `defer func(){defer func(){recover()}();panic("inner")}();panic("outer")`, "", callflow.Panics},
		{"nested recovery after direct ancestor recovery", `defer func(){recover();defer func(){recover()}();panic("inner")}();panic("outer")`, "", callflow.Returns},
		{"nested recovery keeps live store", `flag:=false;defer func(){if !flag||recover()==nil{panic("bad")}}();defer func(){defer func(){recover();flag=true}();panic("inner")}();panic("outer")`, "", callflow.Returns},
		{"recovery postcondition", `flag:=false;defer func(){if !flag{panic("lost store")}}();defer func(){recover();flag=true}();panic("caught")`, "", callflow.Returns},
		{"Goexit postcondition", `flag:=false;defer func(){if !flag{panic("lost store")}}();func(){flag=true;runtime.Goexit()}()`, "", callflow.ExitsGoroutine},
		{"panic Goexit recovery postcondition", `flag:=false;defer func(){if !flag{panic("lost store")}}();defer func(){recover()}();defer func(){flag=true;panic("caught")}();runtime.Goexit()`, "", callflow.ExitsGoroutine},
		{"named result", `if !value(){panic("bad")}`, `func value()(flag bool){defer func(){flag=true}();return false}`, callflow.Returns},
		{"named tuple", `a,b:=value();if a||!b{panic("bad")}`, `func value()(a,b bool){defer func(){a,b=!a,!b}();return true,false}`, callflow.Returns},
		{"unnamed snapshot", `if !value(){panic("bad")}`, `func value()bool{flag:=true;defer func(){flag=false}();return flag}`, callflow.Returns},
		{"named expression snapshot", `if value(){panic("bad")}`, `func value()(result bool){flag:=false;defer func(){flag=true}();return flag}`, callflow.Returns},
		{"named recovered zero", `if value(){panic("bad")}`, `func value()(flag bool){defer func(){recover()}();panic("caught")}`, callflow.Returns},
		{"unnamed recovered zero", `if value(){panic("bad")}`, `func value()bool{defer func(){recover()}();panic("caught")}`, callflow.Returns},
		{"pointer recovered zero", `if value()!=nil{panic("bad")}`, `func value()*int{defer func(){recover()}();panic("caught")}`, callflow.Returns},
		{"interface recovered zero", `if value()!=nil{panic("bad")}`, `func value()any{defer func(){recover()}();panic("caught")}`, callflow.Returns},
		{"store before recovered return", `if !value(){panic("bad")}`, `func value()(flag bool){defer func(){recover()}();flag=true;panic("caught")}`, callflow.Returns},
		{"unbounded registration fixed point", `if value(){panic("maybe")}`, `func next()bool{choose=!choose;return choose};func value()(flag bool){toggle:=func(){flag=!flag};for next(){defer toggle()};return false}`, callflow.Returns | callflow.Panics | callflow.Diverges},
		{"257 registrations", `if !value(){panic("bad")}`, `func value()(flag bool){toggle:=func(){flag=!flag};` + strings.Repeat(`defer toggle();`, 257) + `return false}`, callflow.Returns},
	}
}

func TestUnwindTransfersLiveCellsAndResults(t *testing.T) {
	for _, tc := range unwindValueCases() {
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
