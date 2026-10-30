package taintflow

import (
	"fmt"
	"maps"
	"testing"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func projectionQuery(t testing.TB, name string, outputs int) *query {
	t.Helper()
	b := pdg.NewBuilder()
	evidence := b.Evidence(pdg.Supported, "test")
	origin := b.Origin(pdg.SourceOrigin, "test")
	g := &pdg.Graph{
		Tables: b.Tables, Profile: b.Policy("test", "1"),
		Producer: pdg.Producer{Name: b.Text("test"), Version: b.Text("1"), Language: b.Text("go"), LanguageVersion: b.Known("1.26"), Configuration: b.Text("{}")},
		Sources:  []pdg.Source{{ID: b.Text("source"), Path: b.Text("source.go"), ContentIdentity: b.Known("test")}},
		Function: pdg.Function{ID: b.Text(name), Name: b.Known(name), QualifiedName: b.Known(name), Span: pdg.Span{Source: 1, End: 10}, Interface: evidence, ReferenceCount: pdg.CountFact{Status: pdg.Known}},
		Spans:    []pdg.Span{{Source: 1, End: 10}},
	}
	for _, capability := range pdg.RequiredCapabilities {
		g.Capabilities = append(g.Capabilities, pdg.NamedCapability{Name: b.Text(capability), Evidence: evidence})
	}
	for i := range 3 {
		category := pdg.Computation
		if i == 2 {
			category = pdg.Call
		}
		g.Nodes = append(g.Nodes, pdg.Node{ID: b.Text(fmt.Sprint(i)), Category: category, OriginalKind: b.Known("test"), Span: 1, Evidence: evidence, Provenance: origin})
	}
	g.Nodes[2].Attributes = 1
	if err := pdg.PackAttributes(g, []pdg.Attributes{{CallForm: pdg.DirectCall, Callee: b.Known("callee"), Extensions: b.SingletonExtension("gstyle.go:value-operands", `[1,2]`)}}); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		g.Edges = append(g.Edges, pdg.Edge{ID: b.Text(fmt.Sprint("edge", i)), Source: pdg.Ref(i + 1), Target: 3, Kind: pdg.Data, Subkind: b.Text("source-value"), Position: pdg.Position(i + 1), Evidence: evidence, Provenance: origin})
		g.Function.Inputs = append(g.Function.Inputs, pdg.Parameter{Name: b.Known(fmt.Sprint("input", i)), Type: b.Known("object"), Position: uint32(i), Role: pdg.ParameterRole, Variadic: pdg.BooleanFact{Status: pdg.Known}})
	}
	for i := range outputs {
		g.Function.Outputs = append(g.Function.Outputs, pdg.Parameter{Name: b.Known(fmt.Sprint("output", i)), Type: b.Known("object"), Position: uint32(i), Role: pdg.OutputRole, Variadic: pdg.BooleanFact{Status: pdg.Known}})
	}
	paths, err := bindingflow.New(g)
	if err != nil {
		t.Fatal(err)
	}
	return &query{name: name, key: name, graph: g, paths: paths}
}

func projectionValue(name string, roots ...object) value {
	item := emptyValue()
	item.Data[atom{Source: Site{Function: name, Node: 1}}] = true
	for _, root := range roots {
		item.Objects[root] = true
	}
	return item
}

func projectionFixture(t testing.TB, fanout int, stored, aliased, repeated bool, fields string) (evaluation, *query, summary) {
	t.Helper()
	q := projectionQuery(t, "caller", 4)
	result := emptySummary()
	e := evaluation{query: q, state: emptyState(), result: &result, arguments: map[pdg.Position]value{}}
	base := projectionValue("base")
	for i := range fanout {
		root := object{Input: i + 10}
		child := object{Allocation: fmt.Sprint("child", i)}
		captured := object{Captured: true, Allocation: fmt.Sprint("captured", i)}
		base.Objects[root] = true
		e.state.Repeated[root] = repeated
		if stored {
			item := projectionValue(fmt.Sprint("field", i), child)
			item.Nil, item.Opaque = true, true
			item.Functions["callback"] = true
			e.state.Memory[cellKey(root, "0")] = item
			e.state.Memory[cellKey(child, "1")] = projectionValue("nested", captured)
		}
		e.state.Memory[cellKey(root, "capture:1")] = projectionValue("reference", captured)
		e.state.Memory[cellKey(captured, "value")] = projectionValue("capture")
		e.state.Memory[cellKey(root, "other")] = projectionValue("old other", child)
		e.state.Memory[cellKey(root, "*")] = projectionValue("wildcard")
	}
	e.arguments[1] = base
	e.arguments[2] = projectionValue("other argument", object{Input: 20})
	if aliased {
		e.arguments[2] = base
	}
	maps.Copy(e.state.Nodes, map[slot]value{{Node: 1}: e.arguments[1], {Node: 2}: e.arguments[2]})
	callee := projectionQuery(t, "callee", 4)
	s := emptySummary()
	for i, safe := range []string{"", "html", "sql", ""} {
		item := emptyValue()
		item.Nil, item.Opaque = true, true
		item.Functions["preserved"] = true
		for input := 1; input <= 2; input++ {
			item.Data[atom{Input: input, Fields: fields, SafeFor: safe}] = true
			item.Objects[object{Input: input, Fields: fields}] = true
		}
		item.Data[atom{Source: Site{Function: "callee source", Node: 4}, SafeFor: safe}] = true
		item.Objects[object{Allocation: "callee allocation"}] = true
		s.Results[pdg.Position(i+1)] = item
		s.Writes[cell{Object: object{Input: 1}, Fields: fmt.Sprint("write", i)}] = item
		for _, kind := range []string{"html", "sql", "shell"} {
			s.Sinks[sinkKey{Site: Site{Function: "sink", Node: pdg.Ref(i + 1)}, Kind: kind}] = item.Data
		}
	}
	for _, input := range []int{1, 2} {
		s.Repeated[object{Input: input, Fields: fields}] = true
	}
	s.Repeated[object{Allocation: "callee allocation"}] = true
	s.Writes[cell{Object: object{Input: 1}, Fields: "0"}] = value{Data: dependencies{{Input: 1, Fields: "other"}: true}, Objects: map[object]bool{{Input: 1, Fields: "other"}: true}}
	s.Writes[cell{Object: object{Input: 1}, Fields: "other"}] = value{Data: dependencies{{Input: 1, Fields: "0"}: true}, Objects: map[object]bool{{Input: 1, Fields: "0"}: true}}
	for _, target := range []cell{{Object: object{Input: 1}, Fields: "*"}, {Object: object{Input: 1}, Fields: callbackEffect}, {Object: object{Allocation: "callee allocation"}, Fields: "value"}} {
		s.Writes[target] = s.Results[1]
	}
	return e, callee, s
}

func projectedSummary(e *evaluation) summary {
	result := emptySummary()
	maps.Copy(result.Sinks, e.result.Sinks)
	for position := 1; position <= 4; position++ {
		result.Results[pdg.Position(position)] = e.state.Nodes[slot{Node: 3, Result: pdg.Position(position)}]
	}
	summarizeWrites(&result, map[pdg.Ref]state{1: e.state})
	return result
}

func compareProjection(t *testing.T, cached, uncached *evaluation) {
	t.Helper()
	if !stateEqual(cached.state, uncached.state) {
		t.Fatal("cached projection changes caller state")
	}
	if !summaryEqual(projectedSummary(cached), projectedSummary(uncached)) {
		t.Fatal("cached projection changes caller summary")
	}
}

func TestLocalCallPathCacheMatchesUncachedProjection(t *testing.T) {
	for _, fields := range []string{"", "0", "0/1", "missing", "*", "0/*", callbackEffect, "capture:1/value"} {
		for mask := 0; mask < 16; mask++ {
			t.Run(fmt.Sprintf("%s/%d", fields, mask), func(t *testing.T) {
				fanout, stored, aliased, repeated := 1+mask%2, mask&2 != 0, mask&4 != 0, mask&8 != 0
				cached, callee, s := projectionFixture(t, fanout, stored, aliased, repeated, fields)
				uncached, other, original := projectionFixture(t, fanout, stored, aliased, repeated, fields)
				cached.localCall(3, callee, s)
				uncached.uncachedLocalCall(3, other, original)
				compareProjection(t, &cached, &uncached)
				if cached.argumentPaths != nil || cached.substitutions != nil {
					t.Fatal("substitution caches survive local call")
				}
				if !summaryEqual(s, original) || !maps.EqualFunc(cached.arguments, uncached.arguments, valueEqual) {
					t.Fatal("projection changes arguments or callee summary")
				}
			})
		}
	}
}

func TestLocalCallPathCacheReadsChangedMemoryOnNextCall(t *testing.T) {
	cached, callee, s := projectionFixture(t, 1, true, false, false, "0")
	uncached, other, original := projectionFixture(t, 1, true, false, false, "0")
	for iteration := range 2 {
		if iteration == 1 {
			key := cellKey(object{Input: 10}, "0")
			cached.state.Memory[key] = projectionValue("changed", object{Allocation: "new child"})
			uncached.state.Memory[key] = projectionValue("changed", object{Allocation: "new child"})
		}
		cached.localCall(3, callee, s)
		uncached.uncachedLocalCall(3, other, original)
		compareProjection(t, &cached, &uncached)
	}
	if !cached.state.Nodes[slot{Node: 3, Result: 1}].Data[atom{Source: Site{Function: "changed", Node: 1}}] {
		t.Fatal("second invocation loses changed memory")
	}
}

func TestArgumentPathCacheDoesNotRetainFallbackReads(t *testing.T) {
	for _, mode := range []string{"nil", "partial", "variadic"} {
		t.Run(mode, func(t *testing.T) {
			q := projectionQuery(t, "caller", 4)
			callee := projectionQuery(t, "callee", 4)
			if mode == "variadic" {
				callee.graph.Function.Inputs = callee.graph.Function.Inputs[:1]
				callee.graph.Function.Inputs[0].Variadic.Value = true
			}
			for _, fields := range []string{"", "0"} {
				cached := evaluation{query: q, catalog: map[string]*query{"callee": callee}, state: emptyState(), argumentPaths: map[argumentPath]value{}}
				if mode == "partial" {
					cached.arguments = map[pdg.Position]value{2: projectionValue("fixed")}
				}
				uncached := evaluation{query: q, catalog: cached.catalog, state: emptyState(), arguments: cached.arguments}
				dep := atom{Input: 1, Fields: fields}
				for _, name := range []string{"first", "replacement"} {
					root := object{Allocation: name}
					for _, e := range []*evaluation{&cached, &uncached} {
						e.state.Nodes[slot{Node: 1}] = projectionValue(name, root)
						e.state.Nodes[slot{Node: 2}] = projectionValue("tail")
						e.state.Memory[cellKey(root, "0")] = projectionValue(name)
					}
					got, want := cached.substituteAtom(dep, 3), uncached.uncachedSubstituteAtom(dep, 3)
					if !valueEqual(got, want) || !got.Data[atom{Source: Site{Function: name, Node: 1}}] {
						t.Fatal("fallback retains a stale argument value")
					}
					if mode == "variadic" && fields == "" && !got.Data[atom{Source: Site{Function: "tail", Node: 1}}] {
						t.Fatal("fallback loses the variadic tail")
					}
					if !stateEqual(cached.state, uncached.state) || len(cached.argumentPaths) != 0 {
						t.Fatal("fallback changes state or enters the cache")
					}
				}
			}
		})
	}
}

func TestArgumentPathCacheKeepsExplicitEmptyArgument(t *testing.T) {
	e := evaluation{query: projectionQuery(t, "caller", 4), arguments: map[pdg.Position]value{1: emptyValue()}, argumentPaths: map[argumentPath]value{}, state: emptyState()}
	e.state.Nodes[slot{Node: 1}] = projectionValue("fallback")
	for range 2 {
		if got := e.substituteAtom(atom{Input: 1}, 3); !valueEqual(got, emptyValue()) {
			t.Fatal("explicit empty argument uses fallback")
		}
	}
}

func TestKnownTargetPathCachesMatchUncachedAlternatives(t *testing.T) {
	for _, order := range [][]string{{"fixed", "variadic"}, {"variadic", "fixed"}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			cached, _, s := projectionFixture(t, 1, true, false, false, "0")
			uncached, _, original := projectionFixture(t, 1, true, false, false, "0")
			cached.state.Memory[cellKey(object{Input: 20}, "0")] = projectionValue("second operand", object{Allocation: "second child"})
			uncached.state.Memory[cellKey(object{Input: 20}, "0")] = projectionValue("second operand", object{Allocation: "second child"})
			for _, e := range []*evaluation{&cached, &uncached} {
				e.solver = &summarySolver{callers: map[string][]string{}}
				e.summaries = map[string]summary{"fixed": s, "variadic": original}
			}
			before, otherBefore := cloneState(cached.state), cloneState(uncached.state)
			var joined, otherJoined state
			for _, name := range order {
				callee := projectionQuery(t, name, 4)
				callee.graph.Function.Inputs = callee.graph.Function.Inputs[:1]
				callee.graph.Function.Inputs[0].Variadic.Value = name == "variadic"
				cached.state, uncached.state = cloneState(before), cloneState(otherBefore)
				cached.invokeKnown(3, callee)
				uncached.uncachedInvokeKnown(3, callee)
				compareProjection(t, &cached, &uncached)
				if cached.arguments != nil || cached.argumentPaths != nil || cached.substitutions != nil {
					t.Fatal("known target retains its argument context")
				}
				if name == "variadic" && !cached.state.Nodes[slot{Node: 3, Result: 1}].Data[atom{Source: Site{Function: "second operand", Node: 1}}] {
					t.Fatal("variadic target misses its second argument")
				}
				joined, otherJoined = joinAlternative(joined, cached.state), joinAlternative(otherJoined, uncached.state)
			}
			cached.state, uncached.state = joined, otherJoined
			compareProjection(t, &cached, &uncached)
		})
	}
}

func BenchmarkLocalCallRepeatedPathProjection(b *testing.B) {
	for _, cached := range []bool{false, true} {
		b.Run(fmt.Sprint("cached=", cached), func(b *testing.B) {
			e, callee, s := projectionFixture(b, 8, true, false, false, "0/1")
			before := e.state
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				e.state = cloneState(before)
				result := emptySummary()
				e.result = &result
				if cached {
					e.localCall(3, callee, s)
				} else {
					e.uncachedLocalCall(3, callee, s)
				}
				if len(e.state.Nodes[slot{Node: 3, Result: 1}].Data) == 0 {
					b.Fatal("projection loses dependencies")
				}
			}
		})
	}
}
