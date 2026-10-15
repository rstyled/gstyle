package runtimeguard

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestConstantValueSignedDecimal(t *testing.T) {
	cases := []struct {
		text  string
		value int64
		valid bool
	}{
		{"0", 0, true}, {"-0", 0, true}, {"+17", 17, true}, {"-17", -17, true},
		{"00017", 17, true}, {"-9223372036854775808", -9223372036854775808, true},
		{"9223372036854775807", 9223372036854775807, true},
		{"9223372036854775808", 0, false}, {"-9223372036854775809", 0, false},
		{"", 0, false}, {"-", 0, false}, {"+", 0, false}, {" 1", 0, false},
		{"1_0", 0, false}, {"0x10", 0, false}, {"１２", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			view := constantTestView(t, tc.text, tc.text, "true", "false")
			want := Value{NonNil: true}
			if tc.valid {
				want = Value{NonNil: true, Integer: true, HasLower: true, HasUpper: true,
					Lower: tc.value, Upper: tc.value, HasExtent: true, Extent: tc.value}
			}
			if got := view.constantValue(1); got != want {
				t.Fatalf("constantValue=%+v want=%+v", got, want)
			}
		})
	}
}

func TestConstantValueIndependentFields(t *testing.T) {
	cases := []struct {
		integer, extent, nonNil, nilFact string
		want                             Value
	}{
		{"bad", "-3", "true", "true", Value{NonNil: true, Nil: true, HasExtent: true, Extent: -3}},
		{"4", "bad", "false", "true", Value{Nil: true, Integer: true, HasLower: true, HasUpper: true, Lower: 4, Upper: 4}},
		{"bad", "bad", `"true"`, "TRUE", Value{}},
	}
	for _, tc := range cases {
		view := constantTestView(t, tc.integer, tc.extent, tc.nonNil, tc.nilFact)
		if got := view.constantValue(1); got != tc.want {
			t.Fatalf("input=%+v constantValue=%+v", tc, got)
		}
	}
}

func constantTestView(t *testing.T, integer, extent, nonNil, nilFact string) *graphView {
	t.Helper()
	pool := pdg.NewBuilder()
	entries := []pdg.Extension{}
	for _, fact := range [][2]string{{"integer", integer}, {"literal-extent", extent}, {"non-nil", nonNil}, {"nil", nilFact}} {
		entries = append(entries, pdg.Extension{Name: pool.Text("gstyle.runtime:" + fact[0]), Value: pool.Text(fact[1])})
	}
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Attributes: 1}}}
	if err := pdg.PackAttributes(graph, []pdg.Attributes{{Extensions: entries}}); err != nil {
		t.Fatal(err)
	}
	return &graphView{graph: graph}
}

func TestValueCachedBindingOverridesConstants(t *testing.T) {
	cached := Value{Identity: 91, HasLower: true, Lower: -9, HasExtent: true, Extent: 5, NonNil: true}
	cases := []struct {
		name       string
		attributes pdg.Ref
		key        pdg.Ref
		changing   bool
		state      State
		want       Value
	}{
		{"present attributes", 1, 1, false, State{1: cached}, cached},
		{"absent attributes", 0, 1, false, State{1: cached}, cached},
		{"missing binding", 1, 1, false, State{}, Value{Identity: 2}},
		{"unstable binding", 1, 1, true, State{1: cached}, Value{}},
		{"distinct symbol", 1, 2, false, State{2: cached}, cached},
		{"out of range symbol", 1, ^pdg.Ref(0), false, State{^pdg.Ref(0): cached}, Value{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := constantTestView(t, "42", "99", "true", "true")
			view.graph.Nodes[0].Attributes = tc.attributes
			view.fields = []pdg.Ref{tc.key}
			view.kinds = []string{`"pointer"`, `"pointer"`}
			view.unstable = []bool{tc.changing, tc.changing}
			if got := view.value(1, tc.state); got != tc.want {
				t.Fatalf("value=%+v want=%+v", got, tc.want)
			}
			if got, exists := tc.state[tc.key]; exists && got != cached {
				t.Fatalf("state changed: %+v", tc.state)
			}
		})
	}
}

func TestValueLengthRequiresKnownExactCallee(t *testing.T) {
	for _, name := range []string{"len", "Len", "len ", `"len"`, ""} {
		t.Run(name, func(t *testing.T) {
			view := constantTestView(t, "7", "bad", "false", "false")
			view.fields = []pdg.Ref{0}
			view.operands = [][2]pdg.Ref{{}}
			callee := pdg.Text(len(view.graph.Tables.Strings) + 1)
			view.graph.Tables.Strings = append(view.graph.Tables.Strings, name)
			for status := range 256 {
				view.graph.Attributes[0].Callee = pdg.Fact{Value: callee, Status: pdg.Status(status)}
				want := Value{Integer: true, HasLower: true, HasUpper: true, Lower: 7, Upper: 7,
					Length: pdg.Status(status) == pdg.Known && name == "len"}
				if got := view.value(1, nil); got != want {
					t.Fatalf("status=%d value=%+v want=%+v", status, got, want)
				}
			}
		})
	}
}
