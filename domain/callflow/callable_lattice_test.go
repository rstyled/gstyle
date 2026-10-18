package callflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"reflect"
	"testing"
)

func callableOracleValue(code int) exitValue {
	if code < 5 {
		return scalarValues()[code]
	}
	var names []string
	for i, name := range []string{"p.a", "p.b", "p.c"} {
		if (code-4)&(1<<i) != 0 {
			names = append(names, name)
		}
	}
	return abstractValue(bindingflow.Value{Callables: bindingflow.NewCallableSet(names...)})
}
func TestCallableJoinsAgainstSetOracle(t *testing.T) {
	for a := 0; a < 12; a++ {
		for b := 0; b < 12; b++ {
			want := unknownValue
			if a == b {
				want = callableOracleValue(a)
			} else if a >= 5 && b >= 5 {
				want = callableOracleValue(4 + ((a - 4) | (b - 4)))
			}
			got := decodeTuple(joinReturned(encodeTuple([]exitValue{callableOracleValue(a)}), encodeTuple([]exitValue{callableOracleValue(b)})))
			if len(got) != 1 || got[0] != want {
				t.Fatalf("%d join %d: got %q want %q", a, b, got, want)
			}
		}
	}
}
func TestCallableTupleAndStateKeysRetainEverySlot(t *testing.T) {
	states := map[valueStateID]bool{}
	evaluation := newExitEvaluation(nil, nil)
	for a := 0; a < 12; a++ {
		for b := 0; b < 12; b++ {
			for c := 0; c < 12; c++ {
				values := []exitValue{callableOracleValue(a), callableOracleValue(b), callableOracleValue(c)}
				if got := decodeTuple(encodeTuple(values)); !reflect.DeepEqual(got, values) {
					t.Fatal("tuple loses value")
				}
				key := evaluation.internValues(exitValues{bindings: map[pdg.Ref]exitValue{1: values[0], 2: values[1], 3: values[2]}})
				if states[key] {
					t.Fatalf("distinct states share %d", key)
				}
				states[key] = true
			}
		}
	}
	if len(states) != 1728 {
		t.Fatal("incomplete key oracle")
	}
}
func TestCallableValuesHaveNoByteSizedIdentityLimit(t *testing.T) {
	var names []string
	for i := 0; i < 300; i++ {
		names = append(names, fmt.Sprintf("p.f%03d \" ] / {", i))
	}
	value := abstractValue(bindingflow.Value{Callables: bindingflow.NewCallableSet(names...)})
	tuple := decodeTuple(encodeTuple([]exitValue{value, falseValue, value}))
	got := concreteValue(tuple[2]).Callables.Names()
	if !reflect.DeepEqual(got, names) {
		t.Fatalf("callable identities=%d want %d", len(got), len(names))
	}
	if tuple[1] != falseValue {
		t.Fatal("callable width changes adjacent scalar")
	}
}
