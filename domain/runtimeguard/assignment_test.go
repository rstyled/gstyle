package runtimeguard

import (
	"maps"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func assignmentRawValue() Value {
	return Value{Identity: 17, Lower: -7, Upper: 9, HasLower: true, HasUpper: true,
		NonNil: true, Nil: true, Guarded: true, UpperChecked: true,
		Integer: true, Length: true, BelowLength: true, Extent: 8, HasExtent: true}
}

func TestAssignNonpointerPreservesRecords(t *testing.T) {
	for _, kind := range []string{`"signed"`, `"unsigned"`, `"other"`} {
		view := &graphView{kinds: []string{kind, `"signed"`}}
		value := assignmentRawValue()
		state := State{1: {Identity: 1}, 2: {Lower: -99, HasLower: true}}
		before := maps.Clone(state)
		view.assign(0, 1, value, state)
		if state[1] != value || state[2] != before[2] || len(state) != len(before) {
			t.Fatalf("kind=%s state=%+v", kind, state)
		}
		if value != assignmentRawValue() {
			t.Fatal("assignment changes the caller's Value")
		}
	}
}

func TestAssignmentUnsignedDefault(t *testing.T) {
	view := &graphView{kinds: []string{`"unsigned"`}}
	value := assignmentRawValue()
	value.HasLower = false
	want := value
	want.Lower, want.HasLower = 0, true
	state := State{}
	view.assign(0, 1, value, state)
	if state[1] != want || len(state) != 1 {
		t.Fatalf("state=%+v want=%+v", state, want)
	}
}

func TestSingleValueWriteForms(t *testing.T) {
	cases := []struct {
		category pdg.Category
		form     string
		count    int
		want     bool
	}{
		{pdg.Assignment, "=", 1, true},
		{pdg.Assignment, ":=", 1, true},
		{pdg.Declaration, "", 1, true},
		{pdg.Assignment, "", 1, false},
		{pdg.Other, "other", 1, false},
		{pdg.Assignment, "=", 0, false},
		{pdg.Assignment, "=", 2, false},
		{pdg.Declaration, "", -1, false},
		{pdg.Assignment, "++", 1, false},
		{pdg.Assignment, "--", 1, false},
	}
	for _, tc := range cases {
		if got := singleValueWrite(tc.category, tc.form, tc.count); got != tc.want {
			t.Fatalf("category=%d form=%q count=%d got=%v", tc.category, tc.form, tc.count, got)
		}
	}
}

func TestCompoundTransferDiscardsStaleFacts(t *testing.T) {
	for _, form := range []string{"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", "&^="} {
		pool := pdg.NewBuilder()
		attrs := pdg.Attributes{WriteForm: pool.Text(form), Writes: []pdg.Access{{Symbol: 1}}}
		graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Assignment, Attributes: 1}}, Symbols: make([]pdg.Symbol, 2)}
		if err := pdg.PackAttributes(&graph, []pdg.Attributes{attrs}); err != nil {
			t.Fatal(err)
		}
		view := &graphView{graph: &graph, kinds: []string{`"signed"`, `"signed"`}, tracked: []bool{true, true}}
		state := State{1: assignmentRawValue(), 2: {Identity: 9, Lower: -99, HasLower: true}}
		before := state[2]
		analysis := analysis{view: view}
		analysis.transfer(1, state)
		if state[1] != (Value{}) || state[2] != before || len(state) != 2 {
			t.Fatalf("form=%s state=%+v", form, state)
		}
	}
}

func TestPlainTransferCopiesRhsAndClearsGuardMarker(t *testing.T) {
	for _, form := range []string{"=", ":=", ""} {
		pool := pdg.NewBuilder()
		category := pdg.Assignment
		if form == "" {
			category = pdg.Declaration
		}
		attrs := []pdg.Attributes{
			{WriteForm: pool.Text(form), Writes: []pdg.Access{{Symbol: 1}}},
			{Reads: []pdg.Access{{Symbol: 2}}},
		}
		graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: category, Attributes: 1}, {Category: pdg.Read, Attributes: 2}}, Symbols: make([]pdg.Symbol, 2)}
		if err := pdg.PackAttributes(&graph, attrs); err != nil {
			t.Fatal(err)
		}
		view := &graphView{graph: &graph, kinds: []string{`"signed"`, `"signed"`}, tracked: []bool{true, true}, unstable: []bool{false, false}, fields: make([]pdg.Ref, 2), operands: [][2]pdg.Ref{{2, 0}, {0, 0}}}
		rhs := assignmentRawValue()
		state := State{1: {Identity: 999}, 2: rhs}
		analysis := analysis{view: view}
		analysis.transfer(1, state)
		want := rhs
		want.Guarded = false
		if state[1] != want || state[2] != rhs {
			t.Fatalf("form=%q state=%+v want=%+v", form, state, want)
		}
	}
}
