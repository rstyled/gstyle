package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestBoundEnvironmentsRejectStaleAndIncompleteEvidence(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables, Function: pdg.Function{ID: pool.Text("p.body"), Span: pdg.Span{Source: 1}}, Sources: []pdg.Source{{ContentIdentity: pool.Known("revision")}}}
	summary := ExitSummary{Ordinary: Returns, DeferredPanic: Returns, PreservesCallerMemory: true, invocations: &exitInvocations{graph: graph, captures: []pdg.Ref{1}}}
	calls := &CallSites{}
	for _, target := range []bindingflow.Callable{
		{Name: "p.body", Revision: "old", Cells: []bindingflow.CallableCell{{Symbol: 1, Address: "cell"}}},
		{Name: "p.body", Revision: "revision", Cells: []bindingflow.CallableCell{{Symbol: 2, Address: "cell"}}},
		{Name: "p.body", Revision: "revision", Cells: []bindingflow.CallableCell{{Symbol: 1, Address: "missing"}}},
	} {
		invocation := exitInvocationContext{target: target, values: bindingflow.Evaluation{ClosureCells: map[string]bindingflow.Value{"cell": {Nil: true}}}}
		summaries := ExitSummaries{"p.body": summary}
		if got := calls.targetWithValues("p.body", summaries, invocation); got != UnknownExit {
			t.Fatalf("invalid target proves %v", got)
		}
		if got := calls.targetInvocationMemory("p.body", summaries, invocation); got.preserves || got.confined {
			t.Fatal("invalid target proves memory effects")
		}
	}
}

func TestUnknownCallableRoundTrip(t *testing.T) {
	value := callableValue(bindingflow.CallableSet{})
	if value != unknownValue || callableSet(value).Known() {
		t.Fatal("unknown callable acquires identity")
	}
}
