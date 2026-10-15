package interval

import (
	"math"
	"reflect"
	"testing"
)

func TestSelectFirstRiskyOccurrence(t *testing.T) {
	for length := 0; length <= 6; length++ {
		combinations := 1
		for i := 0; i < length; i++ {
			combinations *= 3
		}
		for keys := 0; keys < combinations; keys++ {
			for mask := 0; mask < 1<<length; mask++ {
				checkSelectedSequence(t, length, keys, mask)
			}
		}
	}
}

func checkSelectedSequence(t *testing.T, length, keys, mask int) {
	t.Helper()
	symbols := []uint32{0, 7, math.MaxUint32}
	var accesses, want []Access
	for i := 0; i < length; i++ {
		lower := int64(0)
		if mask&(1<<i) != 0 {
			lower = -1
		}
		accesses = append(accesses, Access{Node: uint32(i + 1), Symbol: symbols[keys%3], Index: true, UpperChecked: true, Bounds: Range{HasLower: true, Lower: lower, Upper: int64(i + 17)}})
		keys /= 3
	}
	for i, site := range accesses {
		if mask&(1<<i) == 0 {
			continue
		}
		first := true
		for j := 0; j < i; j++ {
			if mask&(1<<j) != 0 && accesses[j].Symbol == site.Symbol {
				first = false
			}
		}
		if first {
			want = append(want, site)
		}
	}
	before := append([]Access(nil), accesses...)
	got := Select(accesses)
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(accesses, before) {
		t.Fatalf("selection changes order, payload or input: in=%+v got=%+v want=%+v", before, got, want)
	}
	if len(got) > 0 {
		got[0].Node = math.MaxUint32
		if !reflect.DeepEqual(accesses, before) {
			t.Fatal("output aliases input")
		}
	}
}

func TestSelectSkipsSafeOccurrencesWithoutMarking(t *testing.T) {
	input := []Access{
		{Node: 1, Symbol: 8, UpperChecked: true},
		{Node: 2, Symbol: 8, Index: true},
		{Node: 3, Symbol: 8, Index: true, UpperChecked: true, Bounds: Range{HasLower: true}},
		{Node: 4, Symbol: 8, Index: true, UpperChecked: true, Bounds: Range{HasLower: true, Lower: 5}},
		{Node: 5, Symbol: 8, Index: true, UpperChecked: true, Bounds: Range{Lower: 100}},
		{Node: 6, Symbol: 8, Index: true, UpperChecked: true, Bounds: Range{HasLower: true, Lower: -3}},
		{Node: 7, Symbol: 9, Index: true, UpperChecked: true},
	}
	if got := Select(input); !reflect.DeepEqual(got, []Access{input[4], input[6]}) {
		t.Fatalf("safe access hides a later risk: %+v", got)
	}
}
