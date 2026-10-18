package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"testing"
)

func TestEnvironmentAlternativesAgainstUnchangedCellOracle(t *testing.T) {
	values := scalarValues()
	checked := 0
	for _, old := range values {
		for left := 0; left <= len(values); left++ {
			for right := 0; right <= len(values); right++ {
				for _, repeated := range []bool{false, true} {
					alternatives := environmentAlternatives{{heap: environmentHeap{}, known: true}, {heap: environmentHeap{}, known: true}}
					expected := []exitValue{old, old}
					for i, index := range []int{left, right} {
						if index != 0 {
							expected[i] = values[index-1]
							alternatives[i].heap["cell"] = environmentCell{Value: expected[i], Repeated: repeated}
						}
					}
					projection := environmentProjection{heap: environmentHeap{}, fresh: map[string]bool{}, known: true}
					for _, alternative := range alternatives {
						projection.merge(alternative)
					}
					projection.retainUntouched(alternatives, bindingflow.Evaluation{ClosureCells: map[string]bindingflow.Value{"cell": concreteValue(old)}, RepeatedCells: map[string]bool{"cell": repeated}})
					heap := environmentHeap{"cell": {Value: old, Repeated: repeated}}
					heap.install(projection, memoryProof{confined: true}, true)
					want := expected[0]
					if expected[0] != expected[1] {
						want = unknownValue
					}
					if got := heap["cell"]; got.Value != want || got.Repeated != repeated {
						t.Fatalf("old=%q alternatives=%v repeated=%v got=%+v want=%q", old, expected, repeated, got, want)
					}
					checked++
				}
			}
		}
	}
	if checked != 360 {
		t.Fatal(checked)
	}
}

func TestMissingAlternativeDoesNotWidenFreshAllocation(t *testing.T) {
	projection := environmentProjection{heap: environmentHeap{"fresh": {Value: falseValue}}, fresh: map[string]bool{"fresh": true}, known: true}
	alternatives := environmentAlternatives{{heap: projection.heap}, {heap: environmentHeap{}}}
	projection.retainUntouched(alternatives, bindingflow.Evaluation{})
	if got := projection.heap["fresh"]; got.Value != falseValue || got.Repeated {
		t.Fatal("conditional allocation loses its value", got)
	}
}
