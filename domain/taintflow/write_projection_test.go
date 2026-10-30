package taintflow

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

type projectionWrite struct {
	key  cell
	item value
}

func collisionProjection(t testing.TB) (evaluation, []projectionWrite) {
	t.Helper()
	result := emptySummary()
	e := evaluation{query: projectionQuery(t, "caller", 0), state: emptyState(), result: &result, arguments: map[pdg.Position]value{}}
	root := object{Allocation: "caller root"}
	other := object{Allocation: "other root"}
	captured := object{Captured: true, Allocation: "captured"}
	local := object{Allocation: "callee allocation"}
	rebased := e.rebaseObject(local, 3)
	e.arguments[1] = projectionValue("first argument", root)
	e.arguments[2] = projectionValue("aliased argument", root, other)
	e.arguments[3] = projectionValue("closure", captured)
	e.arguments[4] = projectionValue("allocation alias", rebased, other)
	for _, target := range []object{root, other, captured} {
		old := projectionValue("original", object{Input: 20})
		old.Functions["original callback"] = true
		old.Nil, old.Opaque = true, true
		e.state.Memory[cellKey(target, "value")] = old
		e.state.Memory[cellKey(target, "*")] = projectionValue("wildcard original")
	}
	e.state.Memory[cellKey(root, "alias")] = projectionValue("alias reference", root)
	e.state.Memory[cellKey(root, "capture:1")] = projectionValue("capture reference", captured)
	keys := []cell{
		{Object: object{Input: 1}, Fields: "value"},
		{Object: object{Input: 2}, Fields: "value"},
		{Object: object{Input: 1, Fields: "alias"}, Fields: "value"},
		{Object: object{Input: 3}, Fields: callbackEffect},
		{Object: object{Input: 1}, Fields: "capture:1/*"},
		{Object: object{Input: 1}, Fields: "*"},
		{Object: object{Input: 2}, Fields: "*"},
		{Object: object{Input: 4}, Fields: "value"},
		{Object: local, Fields: "value"},
		{Object: local, Fields: "value"},
	}
	var writes []projectionWrite
	for index, key := range keys {
		item := projectionValue(fmt.Sprint("write", index), object{Input: 1, Fields: "value"})
		item.Data[atom{Input: 1, Fields: "value", SafeFor: "html"}] = true
		item.Functions[fmt.Sprint("callback", index)] = true
		item.Nil, item.Opaque = index%2 == 0, index%3 == 0
		writes = append(writes, projectionWrite{key: key, item: item})
	}
	return e, writes
}

func TestWeakProjectionMatchesOriginalInEveryCollisionOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for rotation := range 10 {
			t.Run(fmt.Sprintf("reverse=%v/rotation=%d", reverse, rotation), func(t *testing.T) {
				cached, writes := collisionProjection(t)
				original, _ := collisionProjection(t)
				if reverse {
					slices.Reverse(writes)
				}
				writes = append(writes[rotation:], writes[:rotation]...)
				projected := writeProjection{values: map[cell]value{}, weak: map[cell]bool{}}
				want := map[cell]value{}
				before := cloneState(cached.state)
				for index, write := range writes {
					cached.projectWrite(3, write.key, write.item, projected)
					original.uncachedProjectWrite(3, write.key, write.item, want)
					if !stateEqual(cached.state, original.state) {
						t.Fatalf("write %d changes original projection or side effects", index)
					}
				}
				projected.preserveOriginal(cached.state)
				if !maps.EqualFunc(projected.values, want, valueEqual) {
					t.Fatal("batched projection changes final values")
				}
				if !maps.EqualFunc(cached.state.Memory, before.Memory, valueEqual) {
					t.Fatal("projection commits memory before all writes finish")
				}
				if len(projected.weak) == 0 {
					t.Fatal("fixture does not exercise weak collisions")
				}
			})
		}
	}
}

func TestWeakProjectionRetainsWarningsAndReadsEachInvocation(t *testing.T) {
	e, _ := collisionProjection(t)
	s := emptySummary()
	for _, input := range []int{1, 2} {
		s.Writes[cell{Object: object{Input: input}, Fields: "value"}] = emptyValue()
	}
	for _, name := range []string{"first source", "changed source"} {
		source := Site{Function: name, Node: 1}
		key := cellKey(object{Allocation: "caller root"}, "value")
		e.state.Memory[key] = projectionValue(name)
		e.callWrites(3, s)
		sink := sinkKey{Site: Site{Function: "sink", Node: 3}, Kind: "html"}
		e.inheritedSink(3, sink, dependencies{{Input: 1, Fields: "value"}: true})
		got := findings(map[string]summary{"caller": *e.result})
		if !slices.Contains(got, Finding{Source: source, Sink: sink.Site, Kind: sink.Kind}) {
			t.Fatalf("weak clean writes lose warning from %s: %v", name, got)
		}
	}
}

func TestWeakProjectionKeepsAllocationCollisionEffects(t *testing.T) {
	e, writes := collisionProjection(t)
	projected := writeProjection{values: map[cell]value{}, weak: map[cell]bool{}}
	for _, index := range []int{7, 8, 9} {
		write := writes[index]
		e.projectWrite(3, write.key, write.item, projected)
	}
	root := e.rebaseObject(object{Allocation: "callee allocation"}, 3)
	if !e.state.Repeated[root] || !projected.weak[cellKey(root, "value")] {
		t.Fatal("cached weak target skips duplicate allocation side effects")
	}
}

func TestWeakProjectionSeparatesCellsAndStrongWrites(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		e, _ := collisionProjection(t)
		root := object{Allocation: "caller root"}
		e.arguments[2] = e.arguments[1]
		e.state.Repeated[root] = repeated
		s := emptySummary()
		for _, field := range []string{"left", "right"} {
			key := cellKey(root, field)
			e.state.Memory[key] = projectionValue("original " + field)
			for _, input := range []int{1, 2} {
				s.Writes[cell{Object: object{Input: input}, Fields: field}] = projectionValue(fmt.Sprint(field, input))
			}
		}
		e.callWrites(3, s)
		for _, field := range []string{"left", "right"} {
			got := e.state.Memory[cellKey(root, field)]
			if got.Data[atom{Source: Site{Function: "original " + field, Node: 1}}] != repeated {
				t.Fatalf("field=%s repeated=%v changes original-value preservation", field, repeated)
			}
			for _, input := range []int{1, 2} {
				if !got.Data[atom{Source: Site{Function: fmt.Sprint(field, input), Node: 1}}] {
					t.Fatal("aliased write loses an incoming value")
				}
			}
		}
	}
}

func weakProjectionBenchmark(t testing.TB) (evaluation, summary) {
	t.Helper()
	e := evaluation{query: projectionQuery(t, "caller", 0), state: emptyState(), arguments: map[pdg.Position]value{}}
	base, old := emptyValue(), emptyValue()
	for index := range 128 {
		old.Data[atom{Source: Site{Function: "original source", Node: pdg.Ref(index + 1)}}] = true
		old.Objects[object{Allocation: fmt.Sprint("original object", index)}] = true
	}
	for index := range 16 {
		root := object{Allocation: fmt.Sprint("target", index)}
		base.Objects[root] = true
		for field := range 7 {
			e.state.Memory[cellKey(root, fmt.Sprint(field))] = old
		}
	}
	s := emptySummary()
	for index := range 53 {
		e.arguments[pdg.Position(index+1)] = base
		s.Writes[cell{Object: object{Input: index + 1}, Fields: fmt.Sprint(index % 7)}] = projectionValue(fmt.Sprint("write", index))
	}
	return e, s
}

func TestWeakProjectionMatchesOriginalWithManyAliases(t *testing.T) {
	cached, s := weakProjectionBenchmark(t)
	original, _ := weakProjectionBenchmark(t)
	cached.callWrites(3, s)
	original.uncachedCallWrites(3, s)
	if !stateEqual(cached.state, original.state) {
		t.Fatal("aliased writes change original projection")
	}
}

func BenchmarkWeakProjectionCollisions(b *testing.B) {
	for _, cached := range []bool{false, true} {
		b.Run(fmt.Sprint("cached=", cached), func(b *testing.B) {
			e, s := weakProjectionBenchmark(b)
			before := cloneState(e.state)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				e.state = cloneState(before)
				if cached {
					e.callWrites(3, s)
				} else {
					e.uncachedCallWrites(3, s)
				}
			}
			b.ReportMetric(53*16, "targets/op")
			b.ReportMetric(16*7, "unique/op")
		})
	}
}
