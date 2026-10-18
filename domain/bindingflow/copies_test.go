package bindingflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/pdg"
	"reflect"
	"strconv"
	"testing"
)

// Fragments isolate transfer selection. Production New validates the full PDG;
// source-level consumer tests exercise that boundary with complete graphs.
func fragment(t *testing.T, category pdg.Category, configure func(*pdg.Builder, *pdg.Attributes)) *Query {
	t.Helper()
	pool := pdg.NewBuilder()
	attrs := pdg.Attributes{Writes: []pdg.Access{{Symbol: 1, Position: 1}}}
	configure(pool, &attrs)
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Computation, Attributes: 1}, {Category: pdg.Computation, Attributes: 2}, {Category: category, Attributes: 3}}}
	for _, write := range attrs.Writes {
		if write.Symbol == 0 {
			continue
		}
		graph.Definitions = append(graph.Definitions, pdg.Definition{ID: pool.Text("d" + strconv.Itoa(len(graph.Definitions)+1)), Node: 3, Symbol: write.Symbol})
		attrs.Defines = append(attrs.Defines, pdg.Ref(len(graph.Definitions)))
	}
	if err := pdg.PackAttributes(graph, []pdg.Attributes{{}, {}, attrs}); err != nil {
		t.Fatal(err)
	}
	q := &Query{representations: map[pdg.Ref]map[pdg.Position]Representation{}, reaching: make([][]pdg.Edge, len(graph.Nodes)), graph: graph, operands: make([]map[pdg.Position]operand, 3), weak: map[pdg.Ref]bool{}}
	if err := q.indexWeak(); err != nil {
		t.Fatal(err)
	}
	return q
}
func edge(q *Query, source pdg.Ref, position pdg.Position, kind pdg.EdgeKind, subkind string) {
	e := pdg.Edge{Source: source, Target: 3, Position: position, Kind: kind, Subkind: 0}
	// Reuse the query's interned strings by finding the requested fixture token.
	for i, value := range q.graph.Tables.Strings {
		if value == subkind {
			e.Subkind = pdg.Text(i + 1)
		}
	}
	q.index(e)
}
func TestCopySlotsPreserveBlankPositions(t *testing.T) {
	q := fragment(t, pdg.Declaration, func(pool *pdg.Builder, a *pdg.Attributes) {
		pool.Text("source-value")
		a.Writes = []pdg.Access{{Symbol: 1, Position: 1}, {Symbol: 2, Position: 3}}
	})
	edge(q, 1, 1, pdg.Data, "source-value")
	edge(q, 2, 3, pdg.Data, "source-value")
	want := []Write{{Symbol: 1, Position: 1, Source: 1}, {Symbol: 2, Position: 3, Source: 2}}
	if got := q.Writes(3); !reflect.DeepEqual(got, want) {
		t.Fatalf("writes=%+v want %+v", got, want)
	}
}
func TestNonCopyWritesStayUnknown(t *testing.T) {
	for _, tc := range []struct {
		name            string
		category        pdg.Category
		form, construct string
		location        pdg.Ref
	}{
		{"compound", pdg.Assignment, "+=", "", 0}, {"increment", pdg.Assignment, "++", "", 0},
		{"opaque call", pdg.Call, "", "", 0}, {"range binding", pdg.Assignment, ":=", "range-binding", 0},
		{"indirect", pdg.Assignment, "=", "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := fragment(t, tc.category, func(pool *pdg.Builder, a *pdg.Attributes) {
				pool.Text("source-value")
				a.WriteForm = pool.Text(tc.form)
				a.Construct = pool.Text(tc.construct)
				a.Writes[0].Location = tc.location
			})
			edge(q, 1, 1, pdg.Data, "source-value")
			if got := q.Writes(3); len(got) != 1 || got[0].Source != 0 || got[0].Zero {
				t.Fatalf("unsupported transfer=%+v", got)
			}
		})
	}
}
func TestOperandConflictsAndDependencyKinds(t *testing.T) {
	q := fragment(t, pdg.Assignment, func(p *pdg.Builder, a *pdg.Attributes) {
		p.Text("source-value")
		p.Text("call-target")
		p.Text("reaching-definition")
	})
	edge(q, 1, 1, pdg.Data, "source-value")
	edge(q, 1, 1, pdg.Data, "source-value")
	edge(q, 2, 1, pdg.Execution, "source-value")
	edge(q, 2, 1, pdg.Data, "call-target")
	edge(q, 2, 1, pdg.Data, "reaching-definition")
	edge(q, 2, 0, pdg.Data, "source-value")
	if got := q.Operand(3, 1); got != 1 {
		t.Fatalf("other dependencies contaminate value=%d", got)
	}
	edge(q, 2, 1, pdg.Data, "source-value")
	if q.Operand(3, 1) != 0 || q.Writes(3)[0].Source != 0 {
		t.Fatal("conflicting producers become a known copy")
	}
	if q.Operand(3, 2) != 0 {
		t.Fatal("missing slot becomes known")
	}
}
func TestZeroInitializationNeedsExplicitFact(t *testing.T) {
	for _, fact := range []string{"true", "false"} {
		q := fragment(t, pdg.Declaration, func(p *pdg.Builder, a *pdg.Attributes) {
			p.Text("source-value")
			a.Extensions = p.SingletonExtension("gstyle.runtime:zero-initialized", fact)
		})
		got := q.Writes(3)[0]
		if got.Zero != (fact == "true") || got.Source != 0 {
			t.Fatalf("fact %s: %+v", fact, got)
		}
	}
}
func TestQueriesLeaveGraphUnchanged(t *testing.T) {
	q := fragment(t, pdg.Assignment, func(p *pdg.Builder, a *pdg.Attributes) { p.Text("source-value") })
	edge(q, 1, 1, pdg.Data, "source-value")
	before, _ := json.Marshal(q.graph)
	got := q.Writes(3)
	got[0].Symbol = 99
	got[0].Source = 99
	after, _ := json.Marshal(q.graph)
	if string(before) != string(after) || q.Writes(3)[0].Source != 1 {
		t.Fatal("query result mutates borrowed graph")
	}
}
func TestPublicBoundaryRejectsInvalidGraph(t *testing.T) {
	if q, err := New(nil); q != nil || err != nil {
		t.Fatal("nil input should stay unavailable")
	}
	if _, err := New(&pdg.Graph{}); err == nil {
		t.Fatal("invalid graph admitted")
	}
}

func TestDirectResultReadsStaySlotSpecific(t *testing.T) {
	q := fragment(t, pdg.Return, func(p *pdg.Builder, a *pdg.Attributes) {
		a.Reads = []pdg.Access{{Symbol: 1, Position: 1}, {Symbol: 2, Position: 2}}
	})
	if q.ReadSymbol(3, 1) != 1 || q.ReadSymbol(3, 2) != 2 || q.ReadSymbol(3, 3) != 0 {
		t.Fatal("result read positions are lost")
	}
	if got := directRead([]pdg.Access{{Symbol: 1, Position: 1}, {Symbol: 2, Position: 1}}, 1); got != 0 {
		t.Fatal("conflicting result reads become known")
	}
	if got := directRead([]pdg.Access{{Location: 1, Position: 1}}, 1); got != 0 {
		t.Fatal("indirect storage becomes a direct binding")
	}
}

func TestWeakWritesCannotEstablishCopies(t *testing.T) {
	q := fragment(t, pdg.Assignment, func(p *pdg.Builder, a *pdg.Attributes) {
		p.Text("source-value")
		a.Extensions = p.SingletonExtension("gstyle.memory:weak-definitions", `["d1"]`)
	})
	edge(q, 1, 1, pdg.Data, "source-value")
	if got := q.Writes(3); len(got) != 1 || got[0].Source != 0 {
		t.Fatalf("weak write becomes a definite copy: %+v", got)
	}
}
func TestWeakFactsRejectForeignDefinitions(t *testing.T) {
	q := fragment(t, pdg.Assignment, func(p *pdg.Builder, a *pdg.Attributes) {})
	p := pdg.NewBuilder()
	q.graph.Tables = p.Tables
	attrs := q.graph.NodeAttributes(q.graph.Nodes[2])
	extension := pdg.Extension{Name: p.Text("gstyle.memory:weak-definitions"), Value: p.Text(`["other"]`)}
	if err := q.weakFact(3, attrs, extension, map[string]pdg.Ref{}); err == nil {
		t.Fatal("unknown weak definition admitted")
	}
	extension.Value = p.Text(`true`)
	if err := q.weakFact(3, attrs, extension, map[string]pdg.Ref{}); err == nil {
		t.Fatal("malformed weak definition fact admitted")
	}
}

func TestZeroFactCannotOverrideValueOperand(t *testing.T) {
	q := fragment(t, pdg.Declaration, func(p *pdg.Builder, a *pdg.Attributes) {
		p.Text("source-value")
		a.Extensions = p.SingletonExtension("gstyle.runtime:zero-initialized", `true`)
	})
	edge(q, 1, 1, pdg.Data, "source-value")
	if got := q.Writes(3)[0]; got.Zero || got.Source != 0 {
		t.Fatalf("conflicting zero/value facts establish transfer: %+v", got)
	}
}

func TestValueKindsUseTheSameSlotContract(t *testing.T) {
	for _, kind := range []string{"source-value", "binding-value", "scalar-value"} {
		q := fragment(t, pdg.Assignment, func(p *pdg.Builder, a *pdg.Attributes) { p.Text(kind) })
		edge(q, 1, 1, pdg.Data, kind)
		if got := q.Writes(3)[0].Source; got != 1 {
			t.Fatalf("%s producer=%d", kind, got)
		}
	}
}
func TestUnavailableReferencesStayUnknown(t *testing.T) {
	var absent *Query
	if absent.Writes(1) != nil || absent.Operand(1, 1) != 0 || absent.ReadSymbol(1, 1) != 0 {
		t.Fatal("absent query invents facts")
	}
	q := fragment(t, pdg.Return, func(p *pdg.Builder, a *pdg.Attributes) { a.Reads = []pdg.Access{{Symbol: 1}} })
	if q.Writes(0) != nil || q.Writes(4) != nil || q.Operand(3, 0) != 0 || q.ReadSymbol(3, 0) != 0 {
		t.Fatal("unavailable reference invents facts")
	}
}

func TestWeakFactShapeAndOwnership(t *testing.T) {
	for _, value := range []string{`null`, `true`, `{}`, `[1]`} {
		if _, err := decodeWeakFact(value); err == nil {
			t.Fatalf("invalid fact %s admitted", value)
		}
	}
	if ids, err := decodeWeakFact(`[]`); err != nil || len(ids) != 0 {
		t.Fatalf("empty array rejected: %v", err)
	}
	q := fragment(t, pdg.Assignment, func(p *pdg.Builder, a *pdg.Attributes) {})
	pool := pdg.NewBuilder()
	q.graph.Tables = pool.Tables
	attrs := q.graph.NodeAttributes(q.graph.Nodes[2])
	fact := pdg.Extension{Name: pool.Text("gstyle.memory:weak-definitions"), Value: pool.Text(`["d1"]`)}
	if err := q.weakFact(1, attrs, fact, map[string]pdg.Ref{"d1": 1}); err == nil {
		t.Fatal("another owner's weak fact admitted")
	}
	attrs.Defines = nil
	if err := q.weakFact(3, attrs, fact, map[string]pdg.Ref{"d1": 1}); err == nil {
		t.Fatal("unowned weak definition admitted")
	}
}

func TestUnresolvedBindingCannotProveTransfer(t *testing.T) {
	q := fragment(t, pdg.Assignment, func(pool *pdg.Builder, attrs *pdg.Attributes) {
		pool.Text("source-value")
		attrs.Writes[0].UnresolvedReason = pool.Text("uncertain target")
	})
	edge(q, 1, 1, pdg.Data, "source-value")
	if got := q.Writes(3); len(got) != 1 || got[0].Source != 0 || got[0].Zero {
		t.Fatalf("unresolved write proves transfer: %+v", got)
	}
	if got := directRead([]pdg.Access{{Symbol: 1, Position: 1, UnresolvedReason: 1}}, 1); got != 0 {
		t.Fatalf("unresolved read proves identity: %d", got)
	}
}
