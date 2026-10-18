package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestInvocationMemoryRetainsExplicitUnknownWriteBarrier(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Call, Attributes: 1}}}
	attrs := pdg.Attributes{Writes: []pdg.Access{{UnresolvedReason: pool.Text("explicit unknown write")}}}
	if err := pdg.PackAttributes(graph, []pdg.Attributes{attrs}); err != nil {
		t.Fatal(err)
	}
	calls := &CallSites{graph: graph, targets: map[pdg.Ref]callTargets{1: {known: true, names: []string{"p.read"}}}}
	summaries := ExitSummaries{"p.read": {PreservesCallerMemory: true}}
	if got := calls.invocationMemory(summaries, exitInvocationContext{ref: 1}); got.preserves || got.confined {
		t.Fatal("context removes explicit unknown write")
	}
}
