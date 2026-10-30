package taintflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"runtime"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestSubstitutionCacheKeysEveryValuePart(t *testing.T) {
	cached, _, s := projectionFixture(t, 2, true, true, false, "0/1")
	uncached, _, _ := projectionFixture(t, 2, true, true, false, "0/1")
	cached.substitutions = map[valueKey]substitution{}
	base := s.Results[1]
	data, objects, functions, opaque, nilValue := base, base, base, base, base
	data.Data = maps.Clone(base.Data)
	data.Data[atom{Source: Site{Function: "different data", Node: 1}}] = true
	objects.Objects = maps.Clone(base.Objects)
	objects.Objects[object{Allocation: "different object"}] = true
	functions.Functions = maps.Clone(base.Functions)
	functions.Functions["different function"] = true
	opaque.Opaque = false
	nilValue.Nil = false
	for index, item := range []value{base, data, objects, functions, opaque, nilValue, {}, emptyValue()} {
		first := cached.substitute(item, 3)
		second := cached.substitute(item, 3)
		want := uncached.uncachedSubstitute(item, 3)
		if !valueEqual(first, want) || !stateEqual(cached.state, uncached.state) {
			t.Fatalf("value %d changes substitution semantics", index)
		}
		if valueIdentity(first) != valueIdentity(second) {
			t.Fatalf("value %d does not reuse the whole result", index)
		}
		if len(cached.substitutions) != index+1 {
			t.Fatalf("value %d shares a key with a different input", index)
		}
		entry := cached.substitutions[valueIdentity(item)]
		if valueIdentity(entry.Input) != valueIdentity(item) {
			t.Fatal("cache does not retain its input maps")
		}
	}
}

func TestSubstitutionCacheKeepsEqualDistinctMapsSeparate(t *testing.T) {
	e, _, s := projectionFixture(t, 1, true, false, false, "0")
	e.substitutions = map[valueKey]substitution{}
	base := s.Results[1]
	e.substitute(base, 3)
	for range 64 {
		item := base
		item.Data, item.Objects, item.Functions = maps.Clone(base.Data), maps.Clone(base.Objects), maps.Clone(base.Functions)
		e.substitute(item, 3)
	}
	runtime.GC()
	if len(e.substitutions) != 65 {
		t.Fatal("distinct immutable inputs share an identity")
	}
	for key, entry := range e.substitutions {
		if valueIdentity(entry.Input) != key || !valueEqual(entry.Input, base) {
			t.Fatal("cache loses an input map reference")
		}
	}
}

func TestSubstitutionCacheHitsDoNotAllocate(t *testing.T) {
	e, _, s := projectionFixture(t, 2, true, true, false, "0/1")
	e.argumentPaths = map[argumentPath]value{}
	e.substitutions = map[valueKey]substitution{}
	item := s.Results[1]
	want := e.substitute(item, 3)
	allocations := testing.AllocsPerRun(100, func() {
		got := e.substitute(item, 3)
		if valueIdentity(got) != valueIdentity(want) {
			t.Fatal("cache hit reconstructs the result maps")
		}
	})
	if allocations != 0 {
		t.Fatalf("cache hit allocates %g objects", allocations)
	}
}

func TestSubstitutionCacheDoesNotRetainFallbackValues(t *testing.T) {
	for _, mode := range []string{"nil", "partial", "variadic"} {
		for _, part := range []string{"data", "objects"} {
			for _, fields := range []string{"", "0"} {
				t.Run(fmt.Sprintf("%s/%s/%s", mode, part, fields), func(t *testing.T) {
					q, callee := projectionQuery(t, "caller", 4), projectionQuery(t, "callee", 4)
					if mode == "variadic" {
						callee.graph.Function.Inputs = callee.graph.Function.Inputs[:1]
						callee.graph.Function.Inputs[0].Variadic.Value = true
					}
					cached := evaluation{query: q, catalog: map[string]*query{"callee": callee}, state: emptyState(), argumentPaths: map[argumentPath]value{}, substitutions: map[valueKey]substitution{}}
					if mode == "partial" {
						cached.arguments = map[pdg.Position]value{2: projectionValue("fixed")}
					}
					uncached := evaluation{query: q, catalog: cached.catalog, arguments: cached.arguments, state: emptyState()}
					item := value{Data: dependencies{{Input: 2}: true}, Objects: map[object]bool{{Input: 2}: true}}
					if part == "data" {
						item.Data[atom{Input: 1, Fields: fields}] = true
					} else {
						item.Objects[object{Input: 1, Fields: fields}] = true
					}
					var previous value
					for _, name := range []string{"first", "replacement"} {
						root := object{Allocation: name}
						for _, e := range []*evaluation{&cached, &uncached} {
							e.state.Nodes[slot{Node: 1}] = projectionValue(name, root)
							e.state.Nodes[slot{Node: 2}] = projectionValue("tail")
							e.state.Memory[cellKey(root, "0")] = projectionValue(name, root)
						}
						got, want := cached.substitute(item, 3), uncached.uncachedSubstitute(item, 3)
						if !valueEqual(got, want) || valueEqual(got, previous) || len(cached.substitutions) != 0 {
							t.Fatal("fallback value enters the cache or loses a live read")
						}
						previous = got
					}
				})
			}
		}
	}
}

func TestSubstitutionCacheAcceptsExplicitEmptyArgument(t *testing.T) {
	e := evaluation{query: projectionQuery(t, "caller", 4), arguments: map[pdg.Position]value{1: {}}, state: emptyState(), substitutions: map[valueKey]substitution{}}
	e.state.Nodes[slot{Node: 1}] = projectionValue("fallback")
	item := value{Data: dependencies{{Input: 1}: true}, Objects: map[object]bool{{Input: 1}: true}}
	for range 2 {
		if got := e.substitute(item, 3); !valueEqual(got, emptyValue()) {
			t.Fatal("explicit empty argument uses a fallback")
		}
	}
	if len(e.substitutions) != 1 {
		t.Fatal("explicit empty argument cannot enter the cache")
	}
}

func TestSubstitutionOutsideLocalCallReadsCurrentMemory(t *testing.T) {
	e, _, s := projectionFixture(t, 1, true, false, false, "0")
	item := s.Results[1]
	first := e.substitute(item, 3)
	e.state.Memory[cellKey(object{Input: 10}, "0")] = projectionValue("replacement")
	second := e.substitute(item, 3)
	if valueEqual(first, second) || !second.Data[atom{Source: Site{Function: "replacement", Node: 1}}] {
		t.Fatal("substitution outside a local call retains old memory")
	}
	if e.substitutions != nil {
		t.Fatal("substitution creates a cache outside a local call")
	}
}

func TestSubstitutionCacheInvalidatesCallAndCallerIdentity(t *testing.T) {
	cached, callee, s := projectionFixture(t, 1, true, false, false, "0")
	uncached, other, original := projectionFixture(t, 1, true, false, false, "0")
	for _, call := range []pdg.Ref{3, 2, 3} {
		cached.localCall(call, callee, s)
		uncached.uncachedLocalCall(call, other, original)
		compareProjection(t, &cached, &uncached)
	}
	cached.query, uncached.query = projectionQuery(t, "other caller", 4), projectionQuery(t, "other caller", 4)
	cached.localCall(3, callee, s)
	uncached.uncachedLocalCall(3, other, original)
	compareProjection(t, &cached, &uncached)
	if cached.substitutions != nil {
		t.Fatal("substitution cache survives a local call")
	}
}

func TestSubstitutionCacheResultsStayImmutableWhenStored(t *testing.T) {
	e, _, s := projectionFixture(t, 1, true, false, false, "0")
	pool := pdg.NewBuilder()
	pool.Tables = e.query.graph.Tables
	e.query.graph.AttributeTables.Extensions = append(e.query.graph.AttributeTables.Extensions, pool.SingletonExtension("gstyle.memory:kind", `"immutable"`))
	e.query.graph.Attributes[0].Extensions = pdg.Ref(len(e.query.graph.AttributeTables.Extensions))
	e.substitutions = map[valueKey]substitution{}
	item := s.Results[1]
	translated := e.substitute(item, 3)
	before := valueIdentity(translated)
	e.store(slot{Node: 3, Result: 1}, translated)
	if len(e.state.Nodes[slot{Node: 3, Result: 1}].Objects) != 0 {
		t.Fatal("immutable output retains objects")
	}
	again := e.substitute(item, 3)
	if len(again.Objects) == 0 || valueIdentity(again) != before {
		t.Fatal("storing the value mutates the cached object map")
	}
	joined := joinValue(again, projectionValue("joined", object{Allocation: "new"}))
	if valueEqual(joined, again) || valueIdentity(e.substitute(item, 3)) != before {
		t.Fatal("joining a cached value changes it")
	}
}

func repeatedSubstitutionFixture(t testing.TB, fields string) (evaluation, *query, summary) {
	t.Helper()
	e, callee, s := projectionFixture(t, 2, true, true, false, "0/1")
	site, err := json.Marshal(struct {
		Function string
		Node     pdg.Ref
	}{e.query.name, 3})
	if err != nil {
		t.Fatal(err)
	}
	root := object{Allocation: "repeated allocation", Fields: fields}
	root.History, _ = root.History.Extend(string(site))
	item := s.Results[1]
	item.Objects[root] = true
	for position := range callee.graph.Function.Outputs {
		s.Results[pdg.Position(position+1)] = item
	}
	for key := range s.Writes {
		s.Writes[key] = item
	}
	for key := range s.Sinks {
		s.Sinks[key] = item.Data
	}
	child := object{Allocation: "repeated child"}
	e.state.Memory[cellKey(root, "child")] = projectionValue("repeated old", child)
	e.state.Memory[cellKey(child, "next")] = projectionValue("repeated grandchild")
	s.Writes[cell{Object: root, Fields: "child"}] = item
	return e, callee, s
}

func TestSubstitutionCachePreservesRepeatedAllocationEffects(t *testing.T) {
	for _, fields := range []string{"", "0", "*", "0/*"} {
		t.Run(fields, func(t *testing.T) {
			cached, callee, s := repeatedSubstitutionFixture(t, fields)
			uncached, other, original := repeatedSubstitutionFixture(t, fields)
			for range 2 {
				cached.localCall(3, callee, s)
				uncached.uncachedLocalCall(3, other, original)
				compareProjection(t, &cached, &uncached)
				if !summaryEqual(s, original) || cached.substitutions != nil {
					t.Fatal("cache changes or retains the callee summary")
				}
			}
		})
	}
}

func TestSubstitutionCachePreservesEncodingAndCallableValues(t *testing.T) {
	for _, safe := range []string{"", "html", "sql"} {
		cached, _, s := projectionFixture(t, 2, true, true, false, "0")
		uncached, _, _ := projectionFixture(t, 2, true, true, false, "0")
		cached.substitutions = map[valueKey]substitution{}
		item := s.Results[1]
		item.Data = dependencies{{Input: 1, Fields: "0", SafeFor: safe}: true}
		for range 2 {
			got, want := cached.substitute(item, 3), uncached.uncachedSubstitute(item, 3)
			if !valueEqual(got, want) || !got.Nil || !got.Opaque || !got.Functions["preserved"] {
				t.Fatal("cache changes encoding or callable alternatives")
			}
			for dep := range got.Data {
				if dep.SafeFor != safe {
					t.Fatal("cache merges raw and encoded dependencies")
				}
			}
		}
	}
}

func BenchmarkSharedValueSubstitution(b *testing.B) {
	for _, cached := range []bool{false, true} {
		b.Run(fmt.Sprint("cached=", cached), func(b *testing.B) {
			e, _, s := projectionFixture(b, 16, true, true, false, "0/1")
			e.argumentPaths = map[argumentPath]value{}
			e.substitutions = map[valueKey]substitution{}
			item := s.Results[1]
			e.substitute(item, 3)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				var result value
				if cached {
					result = e.substitute(item, 3)
				} else {
					result = e.expandSubstitution(item, 3)
				}
				if len(result.Data) == 0 {
					b.Fatal("substitution loses dependencies")
				}
			}
		})
	}
}
