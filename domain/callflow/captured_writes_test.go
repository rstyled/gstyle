package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestCapturedWriteJoinAgainstTargetChoiceOracle(t *testing.T) {
	for count := 1; count <= 3; count++ {
		combinations := 1
		for range count {
			combinations *= 6
		}
		for _, initial := range scalarValues() {
			for encoded := 0; encoded < combinations; encoded++ {
				var alternatives []map[pdg.Ref]exitValue
				remaining := encoded
				var first exitValue
				uniform := true
				for target := 0; target < count; target++ {
					code := remaining % 6
					remaining /= 6
					writes := map[pdg.Ref]exitValue{}
					concrete := initial
					if code != 0 {
						concrete = scalarValues()[code-1]
						writes[1] = concrete
					}
					alternatives = append(alternatives, writes)
					if target == 0 {
						first = concrete
					} else if concrete != first {
						uniform = false
					}
				}
				want := first
				if !uniform {
					want = unknownValue
				}
				got := joinCaptureWrites(alternatives, map[pdg.Ref]bindingflow.Value{1: concreteValue(initial)})
				// An absent store leaves the entry value. No explicit target stores means
				// the result map stays empty; its default also denotes that same value.
				value, found := got[1]
				if !found {
					value = initial
				}
				if value != want {
					t.Fatalf("targets=%d choices=%d initial=%q got %q want %q", count, encoded, initial, value, want)
				}
			}
		}
	}
}

func TestCapturedWritebackContainsEveryPossibleCellInstance(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, initial := range []exitValue{falseValue, trueValue} {
			for _, post := range []exitValue{falseValue, trueValue} {
				body := &exitBody{facts: bindingflow.Facts{SingleCellSymbols: map[pdg.Ref]bool{1: single}}}
				evaluation := newExitEvaluation(body, nil)
				point := exitPoint{values: evaluation.internValues(exitValues{bindings: map[pdg.Ref]exitValue{1: initial}, cells: map[pdg.Ref]exitValue{1: initial}})}
				next := evaluation.captureWriteback(point, map[pdg.Ref]exitValue{1: post})
				got := evaluation.values[next.values].bindings[1]
				possible := []exitValue{post}
				if !single {
					possible = append(possible, initial)
				}
				for _, actual := range possible {
					if got != unknownValue && got != actual {
						t.Fatalf("single=%v initial=%v post=%v abstraction=%v excludes %v", single, initial, post, got, actual)
					}
				}
				if single && got != post {
					t.Fatal("single cell loses its exact postcondition")
				}
			}
		}
	}
}
