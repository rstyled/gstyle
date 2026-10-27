package taintflow

import (
	"maps"
	"testing"
)

func TestSetContainmentMatchesMembership(t *testing.T) {
	sets := []map[string]bool{nil}
	for mask := range 16 {
		set := map[string]bool{}
		for bit := range 4 {
			if mask&(1<<bit) != 0 {
				set[string(rune('a'+bit))] = true
			}
		}
		sets = append(sets, set)
	}
	for _, to := range sets {
		for _, from := range sets {
			want := true
			for member := range from {
				want = want && to[member]
			}
			if got := setContains(to, from); got != want {
				t.Fatalf("to=%v from=%v: got %v, want %v", to, from, got, want)
			}
		}
	}
	shared := map[string]bool{"first": true}
	alias := shared
	shared["second"] = true
	if !setContains(shared, alias) {
		t.Fatal("shared map loses its members")
	}
}

func BenchmarkSharedValueContainment(b *testing.B) {
	item := sharedBenchmarkValue()
	b.ReportAllocs()
	for b.Loop() {
		if !valueContains(item, item) {
			b.Fatal("shared value does not contain itself")
		}
	}
}

func sharedBenchmarkValue() value {
	item := emptyValue()
	item.Functions["handler"] = true
	for i := range 1024 {
		item.Data[atom{Input: i + 1}] = true
		item.Objects[object{Input: i + 1}] = true
	}
	return item
}

func BenchmarkSharedValueEquality(b *testing.B) {
	item := sharedBenchmarkValue()
	b.ReportAllocs()
	for b.Loop() {
		if !valueEqual(item, item) {
			b.Fatal("shared value is not equal to itself")
		}
	}
}

func TestValueEqualityMatchesMapContents(t *testing.T) {
	items := []value{{}, emptyValue(), sharedBenchmarkValue()}
	for mask := range 16 {
		item := emptyValue()
		for bit := range 4 {
			if mask&(1<<bit) == 0 {
				continue
			}
			item.Data[atom{Input: bit}] = bit%2 == 0
			item.Objects[object{Input: bit}] = bit%3 == 0
			item.Functions[string(rune('a'+bit))] = bit%2 != 0
		}
		item.Nil, item.Opaque = mask&1 != 0, mask&2 != 0
		items = append(items, item)
	}
	for _, left := range items {
		for _, right := range items {
			want := left.Nil == right.Nil && left.Opaque == right.Opaque && maps.Equal(left.Data, right.Data) && maps.Equal(left.Objects, right.Objects) && maps.Equal(left.Functions, right.Functions)
			if valueEqual(left, right) != want {
				t.Fatal("value equality differs from map contents")
			}
		}
	}
}
