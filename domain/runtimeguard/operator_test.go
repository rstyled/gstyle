package runtimeguard

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func operatorView(t *testing.T, status pdg.Status, spelling string) *graphView {
	t.Helper()
	pool := pdg.NewBuilder()
	attrs := []pdg.Attributes{
		{Operator: pdg.Fact{Status: status, Value: pool.Text(spelling)}},
		{Reads: []pdg.Access{{Symbol: 1}}},
		{Extensions: pool.SingletonExtension("gstyle.runtime:integer", "0")},
	}
	graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{
		{Category: pdg.Computation, Attributes: 1},
		{Category: pdg.Read, Attributes: 2},
		{Category: pdg.Computation, Attributes: 3},
	}, Symbols: make([]pdg.Symbol, 1)}
	if err := pdg.PackAttributes(&graph, attrs); err != nil {
		t.Fatal(err)
	}
	return &graphView{graph: &graph, kinds: []string{`"signed"`}, tracked: []bool{true},
		unstable: []bool{false}, fields: make([]pdg.Ref, 3),
		operands: [][2]pdg.Ref{{2, 3}, {0, 0}, {0, 0}}}
}

func TestUnavailableOperatorDoesNotRefine(t *testing.T) {
	for _, status := range []pdg.Status{pdg.Unknown, pdg.Unsupported, pdg.NotApplicable} {
		view := operatorView(t, status, ">=")
		out := view.assume(1, true, State{})
		if out == nil || out[1].HasLower || out[1].HasUpper {
			t.Fatalf("status=%d reason becomes guard: %+v", status, out)
		}
	}
	view := operatorView(t, pdg.Known, ">=")
	out := view.assume(1, true, State{})
	if out == nil || !out[1].HasLower || out[1].Lower != 0 {
		t.Fatalf("known operator loses guard: %+v", out)
	}
}

func TestUnavailableOperatorDoesNotForwardValue(t *testing.T) {
	for _, spelling := range []string{"parentheses", "conversion"} {
		view := operatorView(t, pdg.Unknown, spelling)
		state := State{1: {Lower: 7, Upper: 7, HasLower: true, HasUpper: true, Integer: true, Length: true}}
		if got := view.symbol(1); got != 0 {
			t.Fatalf("reason=%q aliases symbol=%d", spelling, got)
		}
		if got := view.value(1, state); got != (Value{}) {
			t.Fatalf("reason=%q forwards value=%+v", spelling, got)
		}
	}
	view := operatorView(t, pdg.Known, "parentheses")
	state := State{1: {Lower: 7, HasLower: true}}
	if got := view.value(1, state); got != state[1] {
		t.Fatalf("known parentheses loses value: %+v", got)
	}
}

func TestUnavailableOperatorDoesNotDefinePredicate(t *testing.T) {
	for _, status := range []pdg.Status{pdg.Unknown, pdg.Known} {
		view := operatorView(t, status, ">=")
		definitions := make([][]pdg.Ref, 1)
		view.predicateDefinitions(1, definitions)
		var want []pdg.Ref
		if status == pdg.Known {
			want = []pdg.Ref{1}
		}
		if !reflect.DeepEqual(definitions[0], want) {
			t.Fatalf("status=%d definitions=%v want=%v", status, definitions, want)
		}
	}
}

func TestOperatorTextAllStatuses(t *testing.T) {
	pool := pdg.NewBuilder()
	id := pool.Text(">=")
	for status := 0; status <= 255; status++ {
		fact := pdg.Fact{Status: pdg.Status(status), Value: id}
		want := ""
		if fact.Status == pdg.Known {
			want = ">="
		}
		if got := operatorText(pool.Tables, fact); got != want {
			t.Fatalf("status=%d got=%q want=%q", status, got, want)
		}
		if fact.Status != pdg.Known && operatorText(nil, fact) != "" {
			t.Fatalf("status=%d reads unavailable table", status)
		}
	}
	for _, id := range []pdg.Text{0, 2, pdg.Text(^uint32(0))} {
		if got := operatorText(pool.Tables, pdg.Fact{Status: pdg.Known, Value: id}); got != "" {
			t.Fatalf("missing known id=%d got=%q", id, got)
		}
	}
}
