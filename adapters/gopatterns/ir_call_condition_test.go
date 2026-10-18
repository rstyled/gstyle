package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
	"testing"
)

func TestCallContinuationConditionFacts(t *testing.T) {
	for _, tc := range []struct {
		name, body, target string
		want               callflow.Outcomes
	}{
		{"true Boolean arm", `if flag{stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"negated Boolean arm", `if !flag{stop(!flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"named Boolean", `type Flag bool;typed:=Flag(flag);if typed{stop(bool(typed))}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"equality true", `if flag==true{stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"reversed equality", `if true==flag{stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"inequality false", `if flag!=false{stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"reversed inequality", `if false!=flag{stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"false equality arm", `if flag==false{stop(!flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Panics},
		{"equality overwrite", `if flag==true{flag=false;stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Returns},
		{"nil arm", `if pointer==nil{stop(pointer)}`, `func stop(value *int){if value==nil{panic("stop")}}`, callflow.Panics},
		{"nonnil arm", `if pointer!=nil{stop(pointer)}`, `func stop(value *int){if value!=nil{panic("stop")}}`, callflow.Panics},
		{"Boolean overwrite", `if flag{flag=false;stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Returns},
		{"nil overwrite", `if pointer==nil{pointer=new(int);stop(pointer)}`, `func stop(value *int){if value==nil{panic("stop")}}`, callflow.Returns},
		{"Boolean alias write", `if flag{alias:=&flag;*alias=false;stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Returns | callflow.Panics},
		{"unknown pointer write", `if flag{foreign(&flag);stop(flag)}`, `func stop(value bool){if value{panic("stop")}}`, callflow.Returns | callflow.Panics},
		{"joined Boolean arms", `if flag{};stop(flag)`, `func stop(value bool){if value{panic("stop")}}`, callflow.Returns | callflow.Panics},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var foreign func(*bool);func f(flag bool,pointer *int){`+tc.body+`};`+tc.target)
			graphs := map[string]*pdg.Graph{}
			for i := range extraction.Funcs {
				graph := &extraction.Funcs[i].Pdg
				graphs[graph.Tables.Text(graph.Function.ID)] = graph
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			graph := graphs["example.f"]
			queries, err := callflow.NewQueries(graph)
			if err != nil {
				t.Fatal(err)
			}
			outcomes := callflow.OrdinaryCallOutcomes(queries, summaries)
			found := false
			for ref, got := range outcomes {
				if graph.Tables.Text(graph.NodeAttributes(graph.Nodes[ref-1]).Callee.Value) != "example.stop" {
					continue
				}
				found = true
				if got != tc.want {
					t.Errorf("selected-arm call outcomes=%v want %v", got, tc.want)
				}
			}
			if !found {
				t.Fatal("missing stop invocation")
			}
		})
	}
}

func TestCallConditionControlsContinuation(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func stop(value bool){if value{panic("stop")}};func f(flag bool){if flag{stop(flag)};println("after")}`)
	graph := continuationGraph(t, extraction, "example.f")
	for _, edge := range graph.Edges {
		attrs := graph.NodeAttributes(graph.Nodes[edge.Target-1])
		if edge.Kind == pdg.ControlEdge && graph.Tables.Text(attrs.Callee.Value) == "println" && graph.Tables.Text(edge.Outcome) == "false" {
			return
		}
	}
	t.Fatal("post-call continuation lacks its false-arm control dependency")
}

func TestCallConditionFiniteValueOracle(t *testing.T) {
	source := `package example;func stopTrue(x bool){if x{panic("stop")}};func stopFalse(x bool){if !x{panic("stop")}};func stopNil(x *int){if x==nil{panic("stop")}};func stopNonNil(x *int){if x!=nil{panic("stop")}};`
	expected := map[string]callflow.Outcomes{}
	for domain := 0; domain < 2; domain++ {
		for _, guard := range []bool{false, true} {
			for _, panicsOn := range []bool{false, true} {
				for store := 0; store < 3; store++ {
					condition, assignment, call := conditionOracleSyntax(domain, guard, panicsOn, store)
					name := fmt.Sprintf("f%d", len(expected))
					source += fmt.Sprintf("func %s(flag bool,pointer *int){if %s{%s;%s}};", name, condition, assignment, call)
					// Only one of the two concrete input values enters this arm.
					value := guard
					if store != 0 {
						value = store == 2
					}
					want := callflow.Returns
					if value == panicsOn {
						want = callflow.Panics
					}
					expected["example."+name] = want
				}
			}
		}
	}
	extraction, _ := proofExtraction(t, source)
	graphs := map[string]*pdg.Graph{}
	for i := range extraction.Funcs {
		graph := &extraction.Funcs[i].Pdg
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range expected {
		graph := graphs[name]
		queries, err := callflow.NewQueries(graph)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for ref, got := range callflow.OrdinaryCallOutcomes(queries, summaries) {
			callee := graph.Tables.Text(graph.NodeAttributes(graph.Nodes[ref-1]).Callee.Value)
			if !strings.HasPrefix(callee, "example.stop") {
				continue
			}
			found = true
			if got != want {
				t.Errorf("%s: got %v want %v", name, got, want)
			}
		}
		if !found {
			t.Errorf("%s: missing selected call", name)
		}
	}
}

func conditionOracleSyntax(domain int, guard, panicsOn bool, store int) (string, string, string) {
	if domain == 0 {
		condition := "!flag"
		if guard {
			condition = "flag"
		}
		assignments := []string{"", "flag=false", "flag=true"}
		call := "stopFalse(flag)"
		if panicsOn {
			call = "stopTrue(flag)"
		}
		return condition, assignments[store], call
	}
	condition := "pointer!=nil"
	if guard {
		condition = "pointer==nil"
	}
	assignments := []string{"", "pointer=new(int)", "pointer=nil"}
	call := "stopNonNil(pointer)"
	if panicsOn {
		call = "stopNil(pointer)"
	}
	return condition, assignments[store], call
}

func TestBooleanEqualityControlsContinuation(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func stop(value bool){if value{panic("stop")}};func f(flag bool){if flag==true{stop(flag)};println("after")}`)
	graph := continuationGraph(t, extraction, "example.f")
	for _, edge := range graph.Edges {
		attrs := graph.NodeAttributes(graph.Nodes[edge.Target-1])
		if edge.Kind == pdg.ControlEdge && graph.Tables.Text(attrs.Callee.Value) == "println" && graph.Tables.Text(edge.Outcome) == "false" {
			return
		}
	}
	t.Fatal("Boolean equality guard does not control its continuation")
}
