package query_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/pdg/query"
)

func TestAtActualNodesAndAbsence(t *testing.T) {
	nodes := []pdg.Node{{}, {Line: 17, ID: 9, Synthetic: true}}
	before := append([]pdg.Node(nil), nodes...)
	for _, tc := range []struct {
		ref     query.Ref
		want    pdg.Node
		present bool
	}{
		{0, pdg.Node{}, false},
		{1, nodes[0], true},
		{2, nodes[1], true},
		{3, pdg.Node{}, false},
		{query.Ref(math.MaxUint32), pdg.Node{}, false},
	} {
		got, present := query.At(nodes, tc.ref)
		if got != tc.want || present != tc.present {
			t.Fatalf("At(%d) = %+v,%v, want %+v,%v", tc.ref, got, present, tc.want, tc.present)
		}
	}
	if !reflect.DeepEqual(nodes, before) {
		t.Fatal("lookup mutates nodes")
	}
}

func TestAtGenericKinds(t *testing.T) {
	value := 42
	pointers := []*int{nil, &value}
	if got, present := query.At(pointers, 1); got != nil || !present {
		t.Fatal("present nil pointer becomes absence")
	}
	if got, present := query.At(pointers, 2); got != &value || !present {
		t.Fatal("pointer identity changes")
	}
	if got, present := query.At(pointers, 3); got != nil || present {
		t.Fatal("absent pointer becomes present")
	}
	slices := [][]int{{1}, {2, 3}}
	if got, present := query.At(slices, 2); !present || !reflect.DeepEqual(got, slices[1]) {
		t.Fatal("slice record changes")
	}
	maps := []map[string]int{{"answer": 42}}
	if got, present := query.At(maps, 1); !present || got["answer"] != 42 {
		t.Fatal("map record changes")
	}
	functions := []func() int{func() int { return 42 }}
	if got, present := query.At(functions, 1); !present || got() != 42 {
		t.Fatal("function record changes")
	}
	if got, present := query.At(functions, 2); present || got != nil {
		t.Fatal("absent function changes")
	}
	for _, ref := range []query.Ref{0, 1, 2, 3} {
		_, present := query.At(make([]struct{}, 2), ref)
		if present != (ref == 1 || ref == 2) {
			t.Fatalf("zero-size element presence wrong for %d", ref)
		}
	}
	if _, present := query.At([]pdg.Node(nil), 1); present {
		t.Fatal("nil table has an element")
	}
}
