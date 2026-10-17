package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

// Flow fragments isolate the guard proof; adapter tests validate complete graphs.
func TestFlowGuardRequiresEveryEntryPath(t *testing.T) {
	for _, bypass := range []bool{false, true} {
		entry := []pdg.Ref{2}
		if bypass {
			entry = append(entry, 4)
		}
		flow := &pdg.NormalFlow{Entry: 1, Blocks: []pdg.FlowBlock{
			{Successors: entry}, {Condition: 1, Successors: []pdg.Ref{4, 3}}, {}, {Operations: []pdg.Ref{2}, Successors: []pdg.Ref{3}},
		}}
		operations := &OperationEffects{guards: map[pdg.Ref][]flowGuard{}}
		operations.indexFlowGuard(flow, 2, flow.Blocks[1])
		if got := len(operations.guards[2]) > 0; got == bypass {
			t.Fatalf("bypass=%v candidate guarded=%v", bypass, got)
		}
	}
}
