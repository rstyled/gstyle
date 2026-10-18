package bindingflow

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func reachingOrigin(q *Query, definition, target pdg.Ref, position pdg.Position) {
	d := q.graph.Definitions[definition-1]
	q.reaching[target-1] = append(q.reaching[target-1], pdg.Edge{Source: d.Node, Target: target, Definition: definition, Symbol: d.Symbol, Position: position, Kind: pdg.Data})
}
func originCycle(t *testing.T) *Query {
	q := valueQuery(t, "leaf", "copy", "merge", "read", "copy", "read", "copy")
	copyValue(t, q, 2, 1, 1)
	q.graph.Definitions = append(q.graph.Definitions, pdg.Definition{Node: 3, Symbol: 1})
	replaceValueAttrs(t, q, 3, pdg.Merge, pdg.Attributes{Symbols: []pdg.Ref{1}, Defines: []pdg.Ref{2}})
	copyValue(t, q, 5, 1, 4)
	copyValue(t, q, 7, 1, 6)
	reachingOrigin(q, 1, 3, 0)
	reachingOrigin(q, 3, 3, 0)
	reachingOrigin(q, 2, 4, 1)
	reachingOrigin(q, 4, 6, 1)
	return q
}
func TestOriginsResolveProductiveCycles(t *testing.T) {
	q := originCycle(t)
	got := q.OriginsOf(4, Facts{})
	want := []ValueSource{{Node: 1}}
	if !got.Known || !reflect.DeepEqual(got.Sources, want) {
		t.Fatalf("origins=%+v", got)
	}
	// A second, unproductive component cannot borrow a producer from its sibling.
	reachingOrigin(q, 4, 3, 0)
	if got := q.OriginsOf(4, Facts{}); got.Known {
		t.Fatalf("unproductive branch becomes known: %+v", got)
	}
}
func TestOriginsRejectMissingAndWeakDefinitions(t *testing.T) {
	q := originCycle(t)
	q.weak[3] = true
	if got := q.OriginsOf(4, Facts{}); got.Known {
		t.Fatalf("weak write becomes an exact source: %+v", got)
	}
	delete(q.weak, 3)
	q.reaching[3] = nil
	if got := q.OriginsOf(4, Facts{}); got.Known {
		t.Fatalf("missing reaching edge invents source: %+v", got)
	}
}
func TestOriginsKeepTupleResultsSeparate(t *testing.T) {
	q := valueQuery(t, "call", "copy", "read", "read")
	replaceValueAttrs(t, q, 1, pdg.Call, pdg.Attributes{Extensions: []pdg.Extension{{Name: testText(q, "gstyle.memory:kind"), Value: testText(q, `"tuple"`)}}})
	q.graph.Definitions = []pdg.Definition{{Node: 2, Symbol: 1}, {Node: 2, Symbol: 2}}
	replaceValueAttrs(t, q, 2, pdg.Declaration, pdg.Attributes{Writes: []pdg.Access{{Symbol: 1, Position: 1}, {Symbol: 2, Position: 2}}, Defines: []pdg.Ref{1, 2}})
	replaceValueAttrs(t, q, 4, pdg.Read, pdg.Attributes{Reads: []pdg.Access{{Symbol: 2, Position: 1}}})
	q.representations[2] = map[pdg.Position]Representation{1: UnchangedRepresentation, 2: UnchangedRepresentation}
	valueEdge(q, 1, 2, 1)
	reachingOrigin(q, 1, 3, 1)
	reachingOrigin(q, 2, 4, 1)
	for i, ref := range []pdg.Ref{3, 4} {
		got := q.OriginsOf(ref, Facts{})
		if !got.Known || !reflect.DeepEqual(got.Sources, []ValueSource{{Node: 1, Result: pdg.Position(i + 1)}}) {
			t.Fatalf("result %d origins=%+v", i+1, got)
		}
	}
}
func TestOriginResultsAreOwnedAndDeterministic(t *testing.T) {
	q := originCycle(t)
	got := q.OriginsOf(4, Facts{})
	got.Sources[0].Node = 99
	next := q.OriginsOf(4, Facts{})
	if !next.Known || next.Sources[0].Node != 1 {
		t.Fatalf("mutated query result leaks: %+v", next)
	}
}

func TestOriginsRejectUnresolvedCopyTargets(t *testing.T) {
	q := originCycle(t)
	attrs := q.graph.NodeAttributes(q.graph.Nodes[1])
	attrs.Writes = append([]pdg.Access(nil), attrs.Writes...)
	attrs.Writes[0].UnresolvedReason = testText(q, "uncertain target")
	replaceValueAttrs(t, q, 2, pdg.Declaration, attrs)
	if got := q.OriginsOf(4, Facts{}); got.Known {
		t.Fatalf("unresolved write establishes value origin: %+v", got)
	}
}
