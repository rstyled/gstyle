package taintflow

import (
	"fmt"
	"maps"
	"testing"
)

func originalStateJoin(left, right state) state {
	result := cloneState(left)
	maps.Copy(result.Repeated, right.Repeated)
	for key, item := range right.Bindings {
		result.Bindings[key] = joinValue(result.Bindings[key], item)
	}
	for key, item := range right.Nodes {
		result.Nodes[key] = joinValue(result.Nodes[key], item)
	}
	for key, item := range left.Memory {
		result.Memory[key] = joinValue(item, readCell(right, key))
	}
	for key, item := range right.Memory {
		result.Memory[key] = joinValue(item, readCell(left, key))
	}
	return result
}

func TestStateJoinMatchesOriginal(t *testing.T) {
	states := stateJoinFixtures()
	for leftIndex, left := range states {
		for rightIndex, right := range states {
			leftBefore, rightBefore := copyStateValues(left), copyStateValues(right)
			got, want := joinState(left, right), originalStateJoin(left, right)
			if !stateEqual(got, want) {
				t.Fatalf("left=%d right=%d: state join changes facts", leftIndex, rightIndex)
			}
			if !stateEqual(left, leftBefore) || !stateEqual(right, rightBefore) {
				t.Fatalf("left=%d right=%d: state join changes its inputs", leftIndex, rightIndex)
			}
			if got.reads != nil || got.unions != nil || left.unions != nil || right.unions != nil {
				t.Fatal("a state join retains its temporary caches")
			}
		}
	}
}

func stateJoinFixtures() []state {
	states := []state{{}, emptyState()}
	keys := []cell{
		{Object: object{Input: 1}, Fields: "0"},
		{Object: object{Input: 1}, Fields: "*"},
		{Object: object{Input: 1}, Fields: "0/*"},
		{Object: object{Input: 1}, Fields: "0/1"},
		{Object: object{Input: 2}, Fields: "0"},
		{Object: object{Captured: true, Allocation: "cell"}, Fields: "value"},
	}
	for mask := range 1 << len(keys) {
		item := emptyState()
		for index, key := range keys {
			if mask&(1<<index) == 0 {
				continue
			}
			stored := stateJoinValue(index)
			item.Memory[key] = stored
			item.Bindings[1] = stored
			item.Nodes[slot{Node: 1}] = stored
			item.Repeated[key.Object] = true
		}
		states = append(states, item)
	}
	different := emptyState()
	different.Memory[keys[0]] = stateJoinValue(10)
	different.Memory[keys[1]] = stateJoinValue(11)
	different.Memory[keys[3]] = stateJoinValue(12)
	different.Bindings[1] = stateJoinValue(13)
	different.Nodes[slot{Node: 1}] = stateJoinValue(14)
	states = append(states, different)
	return states
}

func stateJoinValue(index int) value {
	item := emptyValue()
	item.Data[atom{Input: index + 1}] = true
	item.Objects[object{Input: index + 1}] = true
	item.Functions[fmt.Sprint(index)] = true
	item.Nil, item.Opaque = index%2 == 0, index%3 == 0
	return item
}

func copyStateValues(input state) state {
	result := cloneState(input)
	for key, item := range result.Bindings {
		result.Bindings[key] = copyJoinValue(item)
	}
	for key, item := range result.Nodes {
		result.Nodes[key] = copyJoinValue(item)
	}
	for key, item := range result.Memory {
		result.Memory[key] = copyJoinValue(item)
	}
	return result
}

func copyJoinValue(item value) value {
	item.Data = maps.Clone(item.Data)
	item.Objects = maps.Clone(item.Objects)
	item.Functions = maps.Clone(item.Functions)
	return item
}

func BenchmarkStateJoinOverlap(b *testing.B) {
	left, right := emptyState(), emptyState()
	for index := range 64 {
		key := cell{Object: object{Input: index/8 + 1}, Fields: fmt.Sprint(index % 8)}
		left.Memory[key], right.Memory[key] = stateJoinValue(index), stateJoinValue(index+1)
	}
	for _, method := range []struct {
		name string
		join func(state, state) state
	}{{"original", originalStateJoin}, {"overlap", joinState}} {
		b.Run(method.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if len(method.join(left, right).Memory) != len(left.Memory) {
					b.Fatal("state join loses stored cells")
				}
			}
		})
	}
}

func BenchmarkStateJoinSharedWildcards(b *testing.B) {
	left, right := emptyState(), emptyState()
	base, overlay, incoming := emptyValue(), emptyValue(), emptyValue()
	for index := range 128 {
		for offset, item := range []value{base, overlay, incoming} {
			item.Data[atom{Input: index + 1 + offset*128}] = true
			item.Objects[object{Input: index + 1 + offset*128}] = true
		}
	}
	for index := range 512 {
		key := cell{Object: object{Input: index/16 + 1}, Fields: fmt.Sprint(index % 16)}
		left.Memory[key], right.Memory[key] = base, incoming
		left.Memory[cellKey(key.Object, "*")] = overlay
	}
	for _, method := range []struct {
		name string
		join func(state, state) state
	}{{"original", originalStateJoin}, {"cached", joinState}} {
		b.Run(method.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if len(method.join(left, right).Memory) != len(left.Memory) {
					b.Fatal("state join loses stored cells")
				}
			}
		})
	}
}
