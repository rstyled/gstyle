package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestInvocationOrderRejectsMalformedOrdinals(t *testing.T) {
	for _, value := range []string{`null`, `0`, `-1`, `1.5`, `"1"`, `[]`, `{}`, `4294967296`, `1 2`} {
		if _, err := decodeInvocationOrder(value); err == nil {
			t.Fatalf("accepted ordinal %s", value)
		}
	}
	if got, err := decodeInvocationOrder(`1`); err != nil || got != 1 {
		t.Fatalf("ordinal=%d error=%v", got, err)
	}
}

func TestInvocationOrderHasOneCallOwner(t *testing.T) {
	operations := &OperationEffects{graph: &pdg.Graph{Nodes: []pdg.Node{{Category: pdg.Call}, {Category: pdg.Call}, {Category: pdg.Computation}}}, orders: map[pdg.Ref]uint32{}}
	if err := operations.indexInvocationOrder(1, `1`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		ref   pdg.Ref
		value string
	}{{1, `2`}, {2, `1`}, {3, `3`}} {
		if err := operations.indexInvocationOrder(tc.ref, tc.value); err == nil {
			t.Fatalf("accepted order %s for owner %d", tc.value, tc.ref)
		}
	}
	if len(operations.orders) != 1 || operations.orders[1] != 1 {
		t.Fatalf("rejected input changed order index: %v", operations.orders)
	}
}
