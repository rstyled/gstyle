package runtimeguard

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestValueEvaluationShortcutsNeedNoGraph(t *testing.T) {
	var view *graphView
	if got := view.value(0, nil); got != (Value{}) {
		t.Fatalf("zero reference value=%+v", got)
	}
	value := Value{Identity: 77, NonNil: true, Integer: true, Lower: -4, HasLower: true, Extent: 8, HasExtent: true}
	if got := view.lengthValue(1, nil, value); got != value {
		t.Fatalf("ordinary value changed: %+v", got)
	}
	if got := view.conversionValue(1, nil, value); got != value {
		t.Fatalf("integer constant changed: %+v", got)
	}
}

func TestConversionValuePreservesOnlyApplicableMetadata(t *testing.T) {
	source := Value{Identity: 77, NonNil: true, Guarded: true, Integer: true, Length: true,
		Lower: -3, Upper: 19, HasLower: true, HasUpper: true, Extent: 8, HasExtent: true, BelowLength: true, UpperChecked: true}
	cases := []struct {
		kind string
		want Value
	}{
		{`"pointer"`, Value{NonNil: true}},
		{`"unsigned"`, Value{HasLower: true, Length: true}},
		{`"signed"`, Value{Length: true}},
		{"pointer", Value{Length: true}},
		{"", Value{Length: true}},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			view := evaluationTestView(t, "conversion", tc.kind)
			view.fields[1] = 1
			state := State{1: source}
			if got := view.conversionValue(1, state, Value{}); got != tc.want {
				t.Fatalf("conversion=%+v want=%+v", got, tc.want)
			}
			if state[1] != source {
				t.Fatal("conversion changes source state")
			}
		})
	}
}

func TestLengthValueUsesExactChildExtent(t *testing.T) {
	candidate := Value{Length: true, Identity: 91, NonNil: true, Guarded: true, Lower: -9, HasLower: true, BelowLength: true}
	cases := []struct {
		source Value
		want   Value
	}{
		{Value{Extent: 7, HasExtent: true, NonNil: true}, Value{Length: true, Integer: true, HasLower: true, HasUpper: true, Lower: 7, Upper: 7}},
		{Value{HasExtent: true}, Value{Length: true, Integer: true, HasLower: true, HasUpper: true}},
		{Value{Extent: 99}, candidate},
	}
	for _, tc := range cases {
		view := evaluationTestView(t, "", "")
		view.fields[1] = 1
		state := State{1: tc.source}
		if got := view.lengthValue(1, state, candidate); got != tc.want {
			t.Fatalf("source=%+v length=%+v want=%+v", tc.source, got, tc.want)
		}
		if state[1] != tc.source {
			t.Fatal("length changes source state")
		}
	}
}

func TestValueParenthesesUsesFirstOperand(t *testing.T) {
	view := evaluationTestView(t, "parentheses", "")
	want := Value{NonNil: true, Integer: true, HasLower: true, HasUpper: true, Lower: 11, Upper: 11, Extent: 7, HasExtent: true}
	if got := view.value(1, nil); got != want {
		t.Fatalf("parenthesized value=%+v want=%+v", got, want)
	}
}

func evaluationTestView(t *testing.T, operator, kind string) *graphView {
	t.Helper()
	pool := pdg.NewBuilder()
	attrs := []pdg.Attributes{
		{Operator: pdg.Fact{Status: pdg.Known, Value: pool.Text(operator)}, Extensions: pool.SingletonExtension("gstyle.runtime:type", kind)},
		{Extensions: []pdg.Extension{
			{Name: pool.Text("gstyle.runtime:integer"), Value: pool.Text("11")},
			{Name: pool.Text("gstyle.runtime:literal-extent"), Value: pool.Text("7")},
			{Name: pool.Text("gstyle.runtime:non-nil"), Value: pool.Text("true")},
		}},
		{Extensions: pool.SingletonExtension("gstyle.runtime:integer", "99")},
	}
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Attributes: 1}, {Attributes: 2}, {Attributes: 3}}}
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
	return &graphView{graph: graph, fields: make([]pdg.Ref, 3), operands: [][2]pdg.Ref{{2, 3}, {}, {}},
		kinds: []string{`"pointer"`}, unstable: []bool{false}}
}
