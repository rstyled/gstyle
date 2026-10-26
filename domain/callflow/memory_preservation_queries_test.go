package callflow

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

type memoryQueriesFixture struct {
	graph *pdg.Graph
	pool  *pdg.Builder
	attrs []pdg.Attributes
}

func newMemoryQueriesFixture(name string) *memoryQueriesFixture {
	pool := pdg.NewBuilder()
	evidence := pool.Evidence(pdg.Supported, "Test operation")
	graph := &pdg.Graph{
		Tables:   pool.Tables,
		Profile:  pool.Policy("fixture", "1"),
		Producer: pdg.Producer{Name: pool.Text("fixture"), Version: pool.Text("1"), Language: pool.Text("go"), LanguageVersion: pool.Known("1.26"), Configuration: pool.Text("{}")},
		Sources:  []pdg.Source{{ID: pool.Text("source"), Path: pool.Text("source.go"), ContentIdentity: pool.Missing(pdg.Unknown, "Test source")}},
		Function: pdg.Function{ID: pool.Text(name), Name: pool.Known(name), QualifiedName: pool.Known(name), Span: pdg.Span{Source: 1}, Interface: evidence, ReferenceCount: pdg.CountFact{Status: pdg.Known}},
	}
	pool.Origin(pdg.SourceOrigin, "Test source")
	for _, name := range pdg.RequiredCapabilities {
		graph.Capabilities = append(graph.Capabilities, pdg.NamedCapability{Name: pool.Text(name), Evidence: evidence})
	}
	return &memoryQueriesFixture{graph: graph, pool: pool}
}

func (fixture *memoryQueriesFixture) operation(category pdg.Category, attrs pdg.Attributes) pdg.Ref {
	ref := pdg.Ref(len(fixture.graph.Nodes) + 1)
	fixture.graph.Nodes = append(fixture.graph.Nodes, pdg.Node{ID: fixture.pool.Text(fmt.Sprint("n", ref)), Category: category, OriginalKind: fixture.pool.Known("fixture"), Attributes: ref, Evidence: 1, Provenance: 1, Synthetic: true})
	fixture.attrs = append(fixture.attrs, attrs)
	return ref
}

func (fixture *memoryQueriesFixture) call(name string) {
	fixture.operation(pdg.Call, pdg.Attributes{CallForm: pdg.DirectCall, Callee: fixture.pool.Known(name)})
}

func (fixture *memoryQueriesFixture) write(role pdg.Role) {
	fixture.graph.Symbols = []pdg.Symbol{{ID: fixture.pool.Text("value"), Scope: fixture.graph.Function.ID, Name: fixture.pool.Known("value"), Type: fixture.pool.Known("bool"), Role: role}}
	fixture.operation(pdg.Assignment, pdg.Attributes{WriteForm: fixture.pool.Text("="), Writes: []pdg.Access{{Symbol: 1, Position: 1}}, Extensions: fixture.pool.SingletonExtension("gstyle.go:operation-effects", `{}`)})
}

func (fixture *memoryQueriesFixture) finish(t *testing.T) *pdg.Graph {
	t.Helper()
	fixture.operation(pdg.Return, pdg.Attributes{})
	operations := make([]pdg.Ref, len(fixture.graph.Nodes))
	for index := range operations {
		operations[index] = pdg.Ref(index + 1)
	}
	fixture.graph.NormalFlow = &pdg.NormalFlow{Entry: 1, Blocks: []pdg.FlowBlock{{Operations: operations, Reachable: true}}}
	if err := pdg.PackAttributes(fixture.graph, fixture.attrs); err != nil {
		t.Fatal(err)
	}
	return fixture.graph
}

func memoryQueries(t *testing.T, graph *pdg.Graph) Queries {
	t.Helper()
	queries, err := NewQueries(graph)
	if err != nil {
		t.Fatal(err)
	}
	return queries
}

func checkMemoryQueries(t *testing.T, prepared map[string]Queries, want map[string]bool) {
	t.Helper()
	got, err := MemoryPreservationWithQueries(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("memory preservation=%v want %v", got, want)
	}
	summaries, err := ExitEffectsWithQueries(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(summaries) {
		t.Fatalf("memory bodies=%d exit bodies=%d", len(got), len(summaries))
	}
	for name, summary := range summaries {
		if value, found := got[name]; !found || value != summary.PreservesCallerMemory {
			t.Fatalf("%s: memory=%v present=%v exit memory=%v", name, value, found, summary.PreservesCallerMemory)
		}
	}
}

func TestMemoryPreservationWithQueriesMatchesExitCallGraphs(t *testing.T) {
	prepared := map[string]Queries{}
	want := map[string]bool{}
	for _, body := range []struct {
		name   string
		calls  []string
		writes bool
		want   bool
	}{
		{"p.pure", nil, false, true},
		{"p.writer", nil, true, false},
		{"p.directPure", []string{"p.pure"}, false, true},
		{"p.transitivePure", []string{"p.directPure"}, false, true},
		{"p.directWriter", []string{"p.writer"}, false, false},
		{"p.transitiveWriter", []string{"p.directWriter"}, false, false},
		{"p.selfPure", []string{"p.selfPure"}, false, true},
		{"p.selfWriter", []string{"p.selfWriter"}, true, false},
		{"p.recursivePureA", []string{"p.recursivePureB"}, false, true},
		{"p.recursivePureB", []string{"p.recursivePureA"}, false, true},
		{"p.recursiveWriterA", []string{"p.recursiveWriterB"}, false, false},
		{"p.recursiveWriterB", []string{"p.recursiveWriterA"}, true, false},
		{"p.recursiveWriterCaller", []string{"p.recursiveWriterA"}, false, false},
		{"p.mixed", []string{"p.pure", "p.writer"}, false, false},
		{"p.builtin", []string{"len"}, false, true},
		{"p.foreign", []string{"foreign.writer"}, false, false},
	} {
		fixture := newMemoryQueriesFixture(body.name)
		if body.writes {
			fixture.write(pdg.GlobalRole)
		}
		for _, name := range body.calls {
			fixture.call(name)
		}
		prepared[body.name] = memoryQueries(t, fixture.finish(t))
		want[body.name] = body.want
	}
	checkMemoryQueries(t, prepared, want)
}

func TestMemoryPreservationWithQueriesClassifiesDirectStorage(t *testing.T) {
	for _, tc := range []struct {
		name string
		role pdg.Role
		want bool
	}{
		{"local", pdg.LocalRole, true},
		{"parameter", pdg.ParameterRole, true},
		{"receiver", pdg.ReceiverRole, true},
		{"output", pdg.OutputRole, true},
		{"global", pdg.GlobalRole, false},
		{"capture", pdg.CaptureRole, false},
		{"other", pdg.OtherRole, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newMemoryQueriesFixture("p.f")
			fixture.write(tc.role)
			prepared := map[string]Queries{"p.f": memoryQueries(t, fixture.finish(t))}
			checkMemoryQueries(t, prepared, map[string]bool{"p.f": tc.want})
		})
	}
}

func TestMemoryPreservationWithQueriesMissingBodies(t *testing.T) {
	checkMemoryQueries(t, nil, map[string]bool{})
	checkMemoryQueries(t, map[string]Queries{}, map[string]bool{})
	checkMemoryQueries(t, map[string]Queries{"p.missing": memoryQueries(t, nil)}, map[string]bool{"p.missing": false})
	for _, supplied := range []bool{false, true} {
		fixture := newMemoryQueriesFixture("p.caller")
		fixture.call("p.missing")
		prepared := map[string]Queries{"p.caller": memoryQueries(t, fixture.finish(t))}
		want := map[string]bool{"p.caller": false}
		if supplied {
			prepared["p.missing"] = Queries{}
			want["p.missing"] = false
		}
		checkMemoryQueries(t, prepared, want)
	}
	fixture := newMemoryQueriesFixture("p.noFlow")
	graph := fixture.finish(t)
	graph.NormalFlow = nil
	checkMemoryQueries(t, map[string]Queries{"p.noFlow": memoryQueries(t, graph)}, map[string]bool{"p.noFlow": false})
}

func TestMemoryPreservationWithQueriesRequiresCompleteQueries(t *testing.T) {
	for _, missing := range []string{"bindings", "calls", "operations"} {
		t.Run(missing, func(t *testing.T) {
			queries := memoryQueries(t, newMemoryQueriesFixture("p.f").finish(t))
			switch missing {
			case "bindings":
				queries.Bindings = nil
			case "calls":
				queries.Calls = nil
			case "operations":
				queries.Operations = nil
			}
			got, err := MemoryPreservationWithQueries(map[string]Queries{"p.f": queries})
			if err != nil || !reflect.DeepEqual(got, map[string]bool{"p.f": false}) {
				t.Fatalf("missing %s: memory=%v error=%v", missing, got, err)
			}
		})
	}
}

func TestMemoryPreservationWithQueriesRequiresExitBodyFlow(t *testing.T) {
	for _, context := range []bool{false, true} {
		fixture := newMemoryQueriesFixture("p.writer")
		fixture.write(pdg.GlobalRole)
		graph := fixture.finish(t)
		exitBody := graph.NormalFlow
		graph.NormalFlow = nil
		graph.ValueFlow = &pdg.NormalFlow{Entry: 1, Blocks: []pdg.FlowBlock{{Operations: []pdg.Ref{2}, Reachable: true}}}
		if context {
			graph.ValueFlow.ExitContext = exitBody
		}
		checkMemoryQueries(t, map[string]Queries{"p.writer": memoryQueries(t, graph)}, map[string]bool{"p.writer": false})
	}
}

func TestMemoryPreservationWithQueriesRequiresOperationProofs(t *testing.T) {
	for _, effects := range []string{"", `{"unknown":true}`} {
		fixture := newMemoryQueriesFixture("p.f")
		attrs := pdg.Attributes{}
		if effects != "" {
			attrs.Extensions = fixture.pool.SingletonExtension("gstyle.go:operation-effects", effects)
		}
		fixture.operation(pdg.Computation, attrs)
		checkMemoryQueries(t, map[string]Queries{"p.f": memoryQueries(t, fixture.finish(t))}, map[string]bool{"p.f": false})
	}
}

func TestMemoryPreservationWithQueriesRetainsUnknownWrites(t *testing.T) {
	for _, category := range []pdg.Category{pdg.Assignment, pdg.Call} {
		fixture := newMemoryQueriesFixture("p.f")
		attrs := pdg.Attributes{Writes: []pdg.Access{{UnresolvedReason: fixture.pool.Text("Unknown storage")}}, Extensions: fixture.pool.SingletonExtension("gstyle.go:operation-effects", `{}`)}
		if category == pdg.Call {
			attrs.CallForm = pdg.DirectCall
			attrs.Callee = fixture.pool.Known("len")
		}
		fixture.operation(category, attrs)
		checkMemoryQueries(t, map[string]Queries{"p.f": memoryQueries(t, fixture.finish(t))}, map[string]bool{"p.f": false})
	}
}

func TestMemoryPreservationWithQueriesRetainsIndirectWrites(t *testing.T) {
	fixture := newMemoryQueriesFixture("p.f")
	fixture.graph.Locations = []pdg.Location{{ID: fixture.pool.Text("storage"), Abstraction: fixture.pool.Text("may-alias"), Description: fixture.pool.Text("Indirect storage"), Evidence: 1}}
	fixture.operation(pdg.Assignment, pdg.Attributes{Writes: []pdg.Access{{Location: 1}}, Extensions: fixture.pool.SingletonExtension("gstyle.go:operation-effects", `{}`)})
	checkMemoryQueries(t, map[string]Queries{"p.f": memoryQueries(t, fixture.finish(t))}, map[string]bool{"p.f": false})
}

func TestMemoryPreservationWithQueriesRejectsAmbiguousTargets(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		fixture := newMemoryQueriesFixture("p.f")
		for range count {
			fixture.operation(pdg.Computation, pdg.Attributes{Extensions: append(fixture.pool.SingletonExtension("gstyle.go:operation-effects", `{}`), pdg.Extension{Name: fixture.pool.Text("gstyle.go:function-value"), Value: fixture.pool.Text(`"len"`)})})
		}
		call := fixture.operation(pdg.Call, pdg.Attributes{CallForm: pdg.IndirectCall})
		for index := range count {
			fixture.graph.Edges = append(fixture.graph.Edges, pdg.Edge{ID: fixture.pool.Text(fmt.Sprint("e", index)), Source: pdg.Ref(index + 1), Target: call, Kind: pdg.Data, Subkind: fixture.pool.Text("call-target"), Evidence: 1, Provenance: 1})
		}
		checkMemoryQueries(t, map[string]Queries{"p.f": memoryQueries(t, fixture.finish(t))}, map[string]bool{"p.f": count == 1})
	}
}

func TestMemoryPreservationQueriesRejectsConflictingEffectProofs(t *testing.T) {
	fixture := newMemoryQueriesFixture("p.f")
	extensions := []pdg.Extension{
		{Name: fixture.pool.Text("gstyle.go:operation-effects"), Value: fixture.pool.Text(`{}`)},
		{Name: fixture.pool.Text("gstyle.go:operation-effects"), Value: fixture.pool.Text(`{"unknown":true}`)},
	}
	fixture.operation(pdg.Computation, pdg.Attributes{Extensions: extensions})
	if _, err := NewQueries(fixture.finish(t)); err == nil {
		t.Fatal("conflicting operation effects establish prepared queries")
	}
}

func TestMemoryPreservationWithQueriesRejectsIdentityMismatch(t *testing.T) {
	prepared := map[string]Queries{
		"p.valid": memoryQueries(t, newMemoryQueriesFixture("p.valid").finish(t)),
		"p.wrong": memoryQueries(t, newMemoryQueriesFixture("p.actual").finish(t)),
	}
	got, err := MemoryPreservationWithQueries(prepared)
	if err == nil || got != nil {
		t.Fatalf("identity mismatch: memory=%v error=%v", got, err)
	}
	summaries, exitErr := ExitEffectsWithQueries(prepared)
	if exitErr == nil || summaries != nil || err.Error() != exitErr.Error() {
		t.Fatalf("memory error=%v exit summaries=%v exit error=%v", err, summaries, exitErr)
	}
}
