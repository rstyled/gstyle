package taintflow

import (
	"fmt"
	"maps"
	"testing"
)

// Use repeated closure over all cells as an oracle, without a traversal queue.
func referenceReachableStore(store state, base value) value {
	result := emptyValue()
	maps.Copy(result.Objects, base.Objects)
	for {
		before := len(result.Objects)
		for root := range maps.Clone(result.Objects) {
			if root.Input != 0 {
				result.Data[atom{Input: root.Input, Fields: appendFields(root.Fields, "*")}] = true
			}
			prefix := root.Fields
			root.Fields = ""
			for key, item := range store.Memory {
				if callbackMarker(key.Fields) || key.Object != root || !fieldPrefix(prefix, key.Fields) {
					continue
				}
				maps.Copy(result.Data, item.Data)
				maps.Copy(result.Objects, item.Objects)
			}
		}
		if before == len(result.Objects) {
			return result
		}
	}
}

func TestReachableStoreMatchesClosure(t *testing.T) {
	roots := []object{{Allocation: "a"}, {Allocation: "b"}, {Input: 1, Fields: "0"}}
	for edges := 0; edges < 1<<9; edges++ {
		store := emptyState()
		for i, root := range roots {
			item := emptyValue()
			item.Data[atom{Source: Site{Function: fmt.Sprint(i)}}] = true
			for j, child := range roots {
				if edges&(1<<(i*3+j)) != 0 {
					item.Objects[child] = true
				}
			}
			store.Memory[cellKey(root, "0")] = item
			store.Memory[cellKey(root, callbackEffect)] = item
		}
		for starts := 0; starts < 8; starts++ {
			base := emptyValue()
			for i, root := range roots {
				if starts&(1<<i) != 0 {
					base.Objects[root] = true
				}
			}
			want := referenceReachableStore(store, base)
			if got := reachableStore(store, base); !valueEqual(got, want) {
				t.Fatalf("edges=%d starts=%d: reachability changes", edges, starts)
			}
		}
	}
}

func BenchmarkReachableSharedGraph(b *testing.B) {
	for _, size := range []int{64, 256, 512} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			store, base := emptyState(), emptyValue()
			for i := 1; i <= size; i++ {
				base.Objects[object{Input: i}] = true
			}
			for root := range base.Objects {
				item := emptyValue()
				maps.Copy(item.Objects, base.Objects)
				item.Data[atom{Input: root.Input}] = true
				store.Memory[cellKey(root, "0")] = item
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				got := reachableStore(store, base)
				if len(got.Objects) != size || len(got.Data) != 2*size {
					b.Fatal("lost reachable facts")
				}
			}
		})
	}
}
