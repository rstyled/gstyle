package bindingflow

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestInputValuesRequireCompleteInventory(t *testing.T) {
	q := valueQuery(t, "leaf", "leaf", "add")
	q.required[3] = []pdg.Position{2, 1}
	valueEdge(q, 1, 3, 1)
	valueEdge(q, 2, 3, 2)
	got, known := q.InputValues(3)
	if !known || !reflect.DeepEqual(got, []pdg.Ref{2, 1}) {
		t.Fatalf("values=%v known=%v", got, known)
	}
	got[0] = 99
	if next, _ := q.InputValues(3); next[0] != 2 {
		t.Fatal("returned inventory aliases query")
	}
	q.required[3] = []pdg.Position{1}
	if _, known := q.InputValues(3); known {
		t.Fatal("unlisted operand ignored")
	}
	q.required[3] = []pdg.Position{1, 2}
	delete(q.operands[2], 2)
	if _, known := q.InputValues(3); known {
		t.Fatal("missing operand accepted")
	}
	valueEdge(q, 2, 3, 2)
	valueEdge(q, 1, 3, 2)
	if _, known := q.InputValues(3); known {
		t.Fatal("conflicting operand accepted")
	}
	delete(q.required, 3)
	if _, known := q.InputValues(3); known {
		t.Fatal("missing inventory accepted")
	}
	q.required[1] = []pdg.Position{}
	if values, known := q.InputValues(1); !known || len(values) != 0 {
		t.Fatalf("empty leaf=%v %v", values, known)
	}
	for _, ref := range []pdg.Ref{0, 4} {
		if _, known := q.InputValues(ref); known {
			t.Fatal("invalid owner accepted")
		}
	}
	var absent *Query
	if _, known := absent.InputValues(1); known {
		t.Fatal("nil query accepted")
	}
}
