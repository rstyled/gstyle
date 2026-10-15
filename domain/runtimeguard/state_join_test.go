package runtimeguard

import (
	"maps"
	"math"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func joinTestView() *graphView {
	return &graphView{kinds: []string{`"signed"`, `"unsigned"`, `"pointer"`, `"signed"`}, unstable: []bool{false, false, false, true}}
}

func TestStateJoinKeyCases(t *testing.T) {
	cases := []struct {
		name string
		p, n State
		want State
	}{
		{"both", State{1: {Lower: -3, Upper: 0, HasLower: true, HasUpper: true, UpperChecked: true}}, State{1: {Lower: 1, Upper: 5, HasLower: true, HasUpper: true}}, State{1: {Lower: -3, Upper: 5, HasLower: true, HasUpper: true, UpperChecked: true}}},
		{"left only signed", State{1: {Lower: 2, HasLower: true}}, State{}, State{1: {}}},
		{"right only unsigned", State{}, State{2: {Lower: 5, Upper: 7, HasLower: true, HasUpper: true}}, State{2: {Lower: 0, HasLower: true}}},
		{"pointer default identity", State{}, State{3: {Identity: 6, NonNil: true}}, State{3: {Identity: 6}}},
		{"unstable cached bound", State{4: {Lower: 7, HasLower: true}}, State{4: {Lower: 8, HasLower: true}}, State{4: {}}},
		{"empty", State{}, State{}, State{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, n := maps.Clone(tc.p), maps.Clone(tc.n)
			got := joinTestView().join(p, n)
			if got == nil || !maps.Equal(got, tc.want) {
				t.Fatalf("got=%+v want=%+v", got, tc.want)
			}
			if !maps.Equal(p, tc.p) || !maps.Equal(n, tc.n) {
				t.Fatal("join mutates an input")
			}
			got[99] = Value{Identity: 99}
			if _, ok := p[99]; ok {
				t.Fatal("output aliases previous")
			}
			if _, ok := n[99]; ok {
				t.Fatal("output aliases next")
			}
		})
	}
}

func TestStateJoinNilCopiesPreserveRawRecords(t *testing.T) {
	raw := State{1: {Identity: math.MaxUint64, Lower: math.MaxInt64, HasLower: true, Integer: true, Extent: 3, HasExtent: true}}
	view := (*graphView)(nil)
	for _, got := range []State{view.join(nil, raw), view.join(raw, nil)} {
		if got == nil || !maps.Equal(got, raw) {
			t.Fatalf("nil branch changes raw records: got=%+v want=%+v", got, raw)
		}
		got[1] = Value{}
		if raw[1].Identity != math.MaxUint64 {
			t.Fatal("nil branch returns an input alias")
		}
	}
	if got := view.join(nil, nil); got == nil || len(got) != 0 {
		t.Fatalf("both nil must produce an allocated empty map: %+v", got)
	}
}

func TestStateJoinIntervalSoundnessAndPreservation(t *testing.T) {
	view := joinTestView()
	states := []State{
		nil, {}, {1: {}},
		{1: {Lower: math.MinInt64, Upper: -1, HasLower: true, HasUpper: true}},
		{1: {Lower: 0, Upper: math.MaxInt64, HasLower: true, HasUpper: true}},
		{1: {Lower: 1, Upper: -1, HasLower: true, HasUpper: true}, 4: {Lower: 8, HasLower: true}},
		{2: {Lower: 5, HasLower: true}, 3: {Identity: 6, NonNil: true}},
	}
	contains := func(v Value, x int64) bool { return (!v.HasLower || v.Lower <= x) && (!v.HasUpper || x <= v.Upper) }
	kinds, unstable := slices.Clone(view.kinds), slices.Clone(view.unstable)
	for _, p := range states {
		for _, n := range states {
			oldp, oldn := maps.Clone(p), maps.Clone(n)
			out := view.join(p, n)
			if out == nil || !maps.Equal(p, oldp) || !maps.Equal(n, oldn) {
				t.Fatal("join changes input maps or returns nil")
			}
			for key := pdg.Ref(1); key <= 4; key++ {
				for _, x := range []int64{math.MinInt64, -2, -1, 0, 1, 2, 8, math.MaxInt64} {
					input := (p != nil && contains(view.binding(key, p), x)) || (n != nil && contains(view.binding(key, n), x))
					if input && !contains(view.binding(key, out), x) {
						t.Fatalf("join excludes represented x=%d key=%d p=%+v n=%+v out=%+v", x, key, p, n, out)
					}
				}
			}
			out[99] = Value{}
			if !maps.Equal(p, oldp) || !maps.Equal(n, oldn) {
				t.Fatal("output aliases an input, including equal input maps")
			}
		}
	}
	if !slices.Equal(view.kinds, kinds) || !slices.Equal(view.unstable, unstable) {
		t.Fatal("join changes graph metadata")
	}
}
