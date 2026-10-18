package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"reflect"
	"slices"
	"testing"
)

func TestEnvironmentRebasingAgainstFiniteSiteSetOracle(t *testing.T) {
	checked := 0
	for length := 0; length <= 5; length++ {
		count := 1
		for range length {
			count *= 3
		}
		for code := 0; code < count; code++ {
			address := ownedJSON(environmentAddress{Owner: "factory", Revision: "rev", Symbol: 2})
			fn := callableValue(bindingflow.NewBoundCallableSet(bindingflow.Callable{Name: "reader", Revision: "rev", Cells: []bindingflow.CallableCell{{Symbol: 1, Address: address}}}))
			projection := environmentProjection{returned: encodeTuple([]exitValue{fn, fn}), heap: environmentHeap{address: {Value: falseValue}}, writes: map[pdg.Ref]exitValue{7: fn}, known: true}
			seen := map[string]bool{}
			repeated := false
			remaining := code
			for range length {
				site := []string{"a", "b", "c"}[remaining%3]
				remaining /= 3
				repeated = repeated || seen[site]
				seen[site] = true
				projection = rebaseEnvironment(projection, nil, site)
				tuple := decodeTuple(projection.returned)
				left := callableSet(tuple[0]).Targets()[0].Cells[0].Address
				right := callableSet(tuple[1]).Targets()[0].Cells[0].Address
				if left != right || callableSet(projection.writes[7]).Targets()[0].Cells[0].Address != left {
					t.Fatal("rebase splits aliases")
				}
				parsed := readOwnedJSON[environmentAddress](left)
				expected := []string{}
				for key := range seen {
					expected = append(expected, key)
				}
				slices.Sort(expected)
				if !reflect.DeepEqual(parsed.Sites, expected) || projection.heap[left].Repeated != repeated {
					t.Fatalf("sites=%v repeated=%v want %v/%v", parsed.Sites, projection.heap[left].Repeated, expected, repeated)
				}
			}
			checked++
		}
	}
	if checked != 364 {
		t.Fatal(checked)
	}
}
func TestEnvironmentWritesAgainstMultiplicityOracle(t *testing.T) {
	for _, old := range scalarValues() {
		for _, next := range scalarValues() {
			for _, repeated := range []bool{false, true} {
				heap := environmentHeap{"cell": {Value: old, Repeated: repeated}}
				writeEnvironmentCell(heap, "cell", next)
				want := next
				if repeated && old != next {
					want = unknownValue
				}
				if got := heap["cell"]; got.Value != want || got.Repeated != repeated {
					t.Fatalf("old=%q next=%q repeated=%v got %+v", old, next, repeated, got)
				}
			}
		}
	}
}
func TestInputCellsKeepIdentityAcrossReturns(t *testing.T) {
	address := ownedJSON(environmentAddress{Owner: "caller", Revision: "rev", Symbol: 1})
	fn := callableValue(bindingflow.NewBoundCallableSet(bindingflow.Callable{Name: "reader", Revision: "rev", Cells: []bindingflow.CallableCell{{Symbol: 1, Address: address}}}))
	heap := environmentHeap{address: {Value: falseValue}}
	projection := rebaseEnvironment(environmentProjection{returned: encodeTuple([]exitValue{fn}), heap: heap}, heap, "callee-site")
	got := callableSet(decodeTuple(projection.returned)[0]).Targets()[0].Cells[0].Address
	if got != address || projection.fresh[address] {
		t.Fatal("callee reallocates caller input")
	}
}
