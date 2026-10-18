package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestOrdinaryCallOutcomesJoinCallerStates(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func stop(flag bool){if flag{panic("stop")}};func f(choose bool){flag:=false;if choose{flag=true};stop(flag)}`)
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
		attrs := graph.NodeAttributes(graph.Nodes[ref-1])
		if graph.Tables.Text(attrs.Callee.Value) != "example.stop" {
			continue
		}
		found = true
		if got != callflow.Returns|callflow.Panics {
			t.Errorf("joined call outcomes=%v", got)
		}
	}
	if !found {
		t.Fatal("missing ordinary stop invocation")
	}
}

// Enumerate the two concrete input paths independently of PDG propagation.
func TestOrdinaryCallOutcomesBooleanAssignmentOracle(t *testing.T) {
	source := "package example;func stop(flag bool){if flag{panic(\"stop\")}};"
	expected := map[string]callflow.Outcomes{}
	for _, initial := range []bool{false, true} {
		for left := 0; left < 3; left++ {
			for right := 0; right < 3; right++ {
				name := fmt.Sprintf("f%d", len(expected))
				source += fmt.Sprintf("func %s(choose bool){flag:=%t;if choose{%s}else{%s};stop(flag)};", name, initial, booleanOracleStore(left), booleanOracleStore(right))
				var want callflow.Outcomes
				for _, arm := range []int{left, right} {
					value := initial
					if arm != 0 {
						value = arm == 2
					}
					if value {
						want |= callflow.Panics
					} else {
						want |= callflow.Returns
					}
				}
				expected["example."+name] = want
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
		queries, err := callflow.NewQueries(graphs[name])
		if err != nil {
			t.Fatal(err)
		}
		outcomes := callflow.OrdinaryCallOutcomes(queries, summaries)
		if len(outcomes) != 1 {
			t.Fatalf("%s: got %d invocation sites", name, len(outcomes))
		}
		for _, got := range outcomes {
			if got != want {
				t.Errorf("%s: got %v want %v", name, got, want)
			}
		}
	}
}

func booleanOracleStore(value int) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("flag=%t", value == 2)
}
