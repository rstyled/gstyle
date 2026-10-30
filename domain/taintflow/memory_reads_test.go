package taintflow

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCallWritesClearsMemoryReadsBeforeExposingNewState(t *testing.T) {
	cached, callee, summary := projectionFixture(t, 2, true, true, false, "0/1")
	original, other, reference := projectionFixture(t, 2, true, true, false, "0/1")
	for _, name := range []string{"first", "replacement"} {
		key := cellKey(object{Input: 10}, "0")
		cached.state.Memory[key] = projectionValue(name)
		original.state.Memory[key] = projectionValue(name)
		cached.localCall(3, callee, summary)
		original.uncachedLocalCall(3, other, reference)
		compareProjection(t, &cached, &original)
		if cached.state.reads != nil || cached.state.unions != nil {
			t.Fatal("memory reads survive the call's write commit")
		}
		expected := projectionValue("after commit")
		cached.state.Memory[key] = expected
		if got := readCell(cached.state, key); !valueEqual(got, readMemoryCell(cached.state, key)) {
			t.Fatal("a completed call leaves a stale memory read")
		}
	}
}

func TestCloningMemoryDropsSnapshotReadCache(t *testing.T) {
	store := emptyState()
	key := cellKey(object{Input: 1}, "value")
	store.Memory[key] = projectionValue("before")
	store.reads = map[cell]value{}
	store.unions = valueUnions{}
	readCell(store, key)
	cloned := cloneState(store)
	cloned.Memory[key] = projectionValue("after")
	if cloned.reads != nil || cloned.unions != nil || !readCell(cloned, key).Data[atom{Source: Site{Function: "after", Node: 1}}] {
		t.Fatal("a cloned state retains reads from the original snapshot")
	}
	if readCell(store, key).Data[atom{Source: Site{Function: "after", Node: 1}}] {
		t.Fatal("updating a clone changes the original snapshot")
	}
}

func TestWeakCommitReadsOriginalWildcardSnapshot(t *testing.T) {
	for range 32 {
		cached, _, summary := projectionFixture(t, 2, true, true, false, "0/1")
		original, _, reference := projectionFixture(t, 2, true, true, false, "0/1")
		cached.callWrites(3, summary)
		original.uncachedCallWrites(3, reference)
		if !stateEqual(cached.state, original.state) {
			t.Fatal("a batched commit reads a newly written wildcard overlay")
		}
	}
}

func BenchmarkSnapshotCellReads(b *testing.B) {
	for _, cached := range []bool{false, true} {
		name := "uncached"
		if cached {
			name = "cached"
		}
		b.Run(name, func(b *testing.B) {
			store := emptyState()
			root := object{Input: 1}
			key := cellKey(root, "value")
			for index := range 512 {
				item := store.Memory[cellKey(root, "*")]
				if item.Data == nil {
					item = emptyValue()
				}
				item.Data[atom{Input: index + 1}] = true
				item.Objects[object{Node: pdg.Ref(index + 1)}] = true
				store.Memory[cellKey(root, "*")] = item
			}
			if cached {
				store.reads = map[cell]value{}
			}
			readCell(store, key)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if len(readCell(store, key).Data) != 513 {
					b.Fatal("snapshot read loses wildcard or symbolic dependencies")
				}
			}
		})
	}
}
