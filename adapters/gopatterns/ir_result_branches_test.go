package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestResultConditionsPruneSharedGraph(t *testing.T) {
	for _, tc := range []struct {
		name, body, helper string
		reachable          bool
	}{
		{"false", `if value(){println("arm")}`, `func value()bool{return false}`, false},
		{"true", `if value(){println("arm")}`, `func value()bool{return true}`, true},
		{"false else", `if value(){}else{println("arm")}`, `func value()bool{return false}`, true},
		{"true else", `if value(){}else{println("arm")}`, `func value()bool{return true}`, false},
		{"copy", `flag:=value();if flag{println("arm")}`, `func value()bool{return false}`, false},
		{"tuple", `a,b:=value();if a||!b{println("arm")}`, `func value()(bool,bool){return false,true}`, false},
		{"nil", `if value()!=nil{println("arm")}`, `func value()*int{return nil}`, false},
		{"argument", `if value(false){println("arm")}`, `func value(flag bool)bool{return flag}`, false},
		{"mixed", `if value(choose){println("arm")}`, `func value(flag bool)bool{return flag}`, true},
		{"overwrite", `flag:=value();flag=true;if flag{println("arm")}`, `func value()bool{return false}`, true},
		{"loop joins", `flag:=value();for choose{if flag{println("arm")};flag=true}`, `func value()bool{return false}`, true},
		{"loop joins false arm", `flag:=value();for choose{if !flag{println("arm")};flag=true}`, `func value()bool{return false}`, true},
		{"unknown prefix joins", `flag:=false;if choose{foreign();flag=true};if flag{println("arm")}`, `var foreign func()`, true},
		{"short circuit", `if value()&&choose{println("arm")}`, `func value()bool{return false}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`};`+tc.helper)
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
				t.Fatalf("arm reachable=%v want %v", reachable, tc.reachable)
			}
		})
	}
}

func assertNoResultExecutionEdges(t *testing.T, graph *pdg.Graph, ref pdg.Ref) {
	t.Helper()
	for _, edge := range graph.Edges {
		if edge.Target == ref && (edge.Kind == pdg.ControlEdge || edge.Kind == pdg.Execution) {
			t.Fatal("unreachable operation retains execution/control edge")
		}
	}
}

func TestOrdinaryResultPruningRetainsInheritedExitContext(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func f(){if recover()!=nil{panic("again")}}`)
	graph := continuationGraph(t, extraction, "example.f")
	if graph.ValueFlow.ExitContext == nil {
		t.Fatal("missing original exit context")
	}
	summaries, err := callflow.ExitEffects(map[string]*pdg.Graph{"example.f": graph})
	if err != nil {
		t.Fatal(err)
	}
	got := summaries["example.f"]
	if got.Ordinary != callflow.Returns || got.DeferredPanic != callflow.Panics {
		t.Fatalf("ordinary=%v inherited=%v", got.Ordinary, got.DeferredPanic)
	}
}
