package taintflow

import (
	"fmt"
	"testing"
)

func TestPathComponentMatchesValueJoins(t *testing.T) {
	store, base := emptyState(), emptyValue()
	for i := 1; i <= 32; i++ {
		root := object{Input: i}
		base.Objects[root] = true
		if i%3 == 0 {
			continue // Keep symbolic reads in the union too.
		}
		item := emptyValue()
		item.Data[atom{Input: i}] = true
		item.Objects[object{Allocation: fmt.Sprint(i)}] = true
		item.Functions[fmt.Sprint(i%4)] = true
		item.Nil, item.Opaque = i%2 == 0, i%5 == 0
		store.Memory[cellKey(root, "0")] = item
		store.Memory[cellKey(root, "*")] = item
	}
	for _, field := range []string{"0", "1", "*", callbackEffect} {
		t.Run(field, func(t *testing.T) {
			want := emptyValue()
			switch field {
			case "*":
				want = wildcardValue(store, base)
			case callbackEffect:
				want = callbackValue(store, base)
			default:
				for root := range base.Objects {
					want = joinValue(want, readCell(store, cellKey(root, field)))
				}
			}
			if got := pathComponent(store, base, field); !valueEqual(got, want) {
				t.Fatal("path read changes the union")
			}
		})
	}
	if got := pathComponent(store, emptyValue(), "0"); !valueEqual(got, emptyValue()) {
		t.Fatal("empty path read contains facts")
	}
}

func TestPathUnionKeepsStoredMaps(t *testing.T) {
	store, base := emptyState(), emptyValue()
	for i := 1; i <= 2; i++ {
		root := object{Input: i}
		base.Objects[root] = true
		item := emptyValue()
		item.Data[atom{Input: i}] = true
		item.Objects[root] = true
		item.Functions[fmt.Sprint(i)] = true
		store.Memory[cellKey(root, "0")] = item
	}
	got := pathComponent(store, base, "0")
	clear(got.Data)
	clear(got.Objects)
	clear(got.Functions)
	for _, item := range store.Memory {
		if len(item.Data) != 1 || len(item.Objects) != 1 || len(item.Functions) != 1 {
			t.Fatal("path union changes stored maps")
		}
	}
}

func BenchmarkPathComponentFanout(b *testing.B) {
	for _, size := range []int{64, 256, 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			base := emptyValue()
			for i := 1; i <= size; i++ {
				base.Objects[object{Input: i}] = true
			}
			store := emptyState()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				got := pathComponent(store, base, "0")
				if len(got.Objects) != size || len(got.Data) != size {
					b.Fatal("lost dependencies")
				}
			}
		})
	}
}
