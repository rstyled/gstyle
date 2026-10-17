package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestCanonicalValueFactsDriveExitQueries(t *testing.T) {
	for _, tc := range []struct {
		name, body          string
		ordinary, inherited callflow.Outcomes
	}{
		{"false recursive branch", `if false{f()}`, callflow.Returns, callflow.Returns},
		{"named false constant", `const disabled=false;if disabled{f()}`, callflow.Returns, callflow.Returns},
		{"folded constant", `if 2+2==4{return};f()`, callflow.Returns, callflow.Returns},
		{"identity conversion", `if any(recover())==nil{panic("empty")}`, callflow.Panics, callflow.RecoversInherited},
		{"nil conversion", `recover();value:=any(nil);if value!=nil{panic("bad")}`, callflow.Returns, callflow.RecoversInherited},
		{"boxed pointer conversion", `recover();var pointer *int;value:=any(pointer);if value==nil{panic("bad")}`, callflow.Returns, callflow.RecoversInherited},
		{"interface conversion", `recover();var err error;value:=any(err);if value!=nil{panic("bad")}`, callflow.Returns, callflow.RecoversInherited},
		{"named pointer conversion", `recover();type P *int;var pointer *int;value:=P(pointer);if value!=nil{panic("bad")}`, callflow.Returns, callflow.RecoversInherited},
		{"named slice conversion", `recover();type S []int;var source []int;value:=S(source);if value!=nil{panic("bad")}`, callflow.Returns, callflow.RecoversInherited},
		{"nonpreserving conversion stays unknown", `recover();var source []byte;text:=string(source);value:=[]byte(text);if value==nil{panic("unknown")}`, callflow.Returns | callflow.Panics, callflow.RecoversInherited | callflow.Panics},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := recoveryValueGraph(t, `package p;func f(){`+tc.body+`}`, pdg.NewBuilder())
			name := graph.Tables.Text(graph.Function.ID)
			values, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph})
			if err != nil {
				t.Fatal(err)
			}
			got := values[name]
			if got.Ordinary != tc.ordinary || got.DeferredPanic != tc.inherited {
				t.Fatalf("ordinary=%v inherited=%v want %v/%v", got.Ordinary, got.DeferredPanic, tc.ordinary, tc.inherited)
			}
		})
	}
}
