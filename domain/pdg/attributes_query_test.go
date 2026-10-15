package pdg

import (
	"reflect"
	"slices"
	"testing"
)

func TestNodeAttributesAbsence(t *testing.T) {
	var absent *Graph
	if got := absent.NodeAttributes(Node{}); !reflect.DeepEqual(got, Attributes{}) {
		t.Fatalf("zero reference dereferences graph: %+v", got)
	}
	for _, graph := range []Graph{{}, {Attributes: []AttributeSet{}}} {
		for _, ref := range []Ref{0, 1, Ref(^uint32(0))} {
			if got := graph.NodeAttributes(Node{Attributes: ref}); !reflect.DeepEqual(got, Attributes{}) {
				t.Fatalf("ref=%d yields attributes: %+v", ref, got)
			}
		}
	}
	backing := []AttributeSet{{Construct: 1}, {Construct: 2}, {Construct: 3}}
	graph := Graph{Attributes: backing[:1]}
	if got := graph.NodeAttributes(Node{Attributes: 2}); !reflect.DeepEqual(got, Attributes{}) {
		t.Fatalf("spare capacity becomes a record: %+v", got)
	}
}

func TestNodeAttributesExpansionAndPreservation(t *testing.T) {
	graph := attributeQueryGraph()
	before := snapshotAttributeGraph(graph)
	first := Attributes{Symbols: []Ref{1, 2, Ref(^uint32(0))}, Defines: []Ref{8},
		Reads: []Access{{Symbol: 2, UnresolvedReason: 3, Position: 4}}, Writes: []Access{{Location: 7, Position: 2}},
		Operator: Fact{Status: Known, Value: 31}, Literal: Fact{Status: Unknown, Value: 32}, Callee: Fact{Status: Unsupported, Value: 33},
		Construct: 4, WriteForm: 5, Description: 6, Position: Position(^uint32(0)), CallForm: UnknownCall, ExitKind: OtherExit,
		Extensions: []Extension{{Name: 11, Value: 12}}, Outcomes: []Text{4, Text(^uint32(0)), 0, 23}}
	second := Attributes{Symbols: []Ref{8}, Defines: []Ref{1, 2, Ref(^uint32(0))},
		Reads: first.Writes, Writes: first.Reads, Operator: Fact{Status: NotApplicable, Value: 41},
		Literal: Fact{Status: Known, Value: 42}, Callee: Fact{Status: Unknown, Value: 43},
		Construct: 14, WriteForm: 15, Description: 16, Position: 1, CallForm: DirectCall, ExitKind: NormalExit,
		Extensions: []Extension{{Name: 9, Value: 10}}}
	for i, want := range []Attributes{first, second} {
		node := Node{Attributes: Ref(i + 1), Line: 777, Category: Read, Synthetic: true}
		original := node
		got := graph.NodeAttributes(node)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ref=%d got=%+v want=%+v", i+1, got, want)
		}
		if node != original || !reflect.DeepEqual(graph, before) {
			t.Fatal("lookup changes its inputs")
		}
		set := graph.Attributes[i]
		if &got.Symbols[0] != &graph.AttributeTables.References[set.Symbols-1][0] ||
			&got.Reads[0] != &graph.AttributeTables.Accesses[set.Reads-1][0] ||
			&got.Extensions[0] != &graph.AttributeTables.Extensions[set.Extensions-1][0] {
			t.Fatal("expansion copies a shared row")
		}
	}
}

func TestOutcomesCopyAndEmptyRows(t *testing.T) {
	graph := attributeQueryGraph()
	for _, ref := range []Ref{0, 4, 5, 6, Ref(^uint32(0))} {
		if got := graph.outcomes(ref); got != nil {
			t.Fatalf("ref=%d got=%v want nil", ref, got)
		}
	}
	before := snapshotAttributeGraph(graph)
	got := graph.outcomes(3)
	if !slices.Equal(got, []Text{4, Text(^uint32(0)), 0, 23}) || !reflect.DeepEqual(graph, before) {
		t.Fatalf("outcomes loses words or changes source: %v", got)
	}
	got[0] = 99
	if !reflect.DeepEqual(graph, before) {
		t.Fatal("outcomes result aliases its source")
	}
}

func attributeQueryGraph() Graph {
	return Graph{Profile: 99, Producer: Producer{Name: 6}, Function: Function{ID: 5}, Conformance: Conformance{Status: NonConformant},
		Attributes: []AttributeSet{
			{Symbols: 1, Defines: 2, Reads: 1, Writes: 2, Extensions: 2, Outcomes: 3,
				Operator: Fact{Status: Known, Value: 31}, Literal: Fact{Status: Unknown, Value: 32}, Callee: Fact{Status: Unsupported, Value: 33},
				Construct: 4, WriteForm: 5, Description: 6, Position: Position(^uint32(0)), CallForm: UnknownCall, ExitKind: OtherExit},
			{Symbols: 2, Defines: 1, Reads: 2, Writes: 1, Extensions: 1, Outcomes: 4,
				Operator: Fact{Status: NotApplicable, Value: 41}, Literal: Fact{Status: Known, Value: 42}, Callee: Fact{Status: Unknown, Value: 43},
				Construct: 14, WriteForm: 15, Description: 16, Position: 1, CallForm: DirectCall, ExitKind: NormalExit},
		}, AttributeTables: AttributeTables{
			References: [][]Ref{{1, 2, Ref(^uint32(0))}, {8}, {4, Ref(^uint32(0)), 0, 23}, make([]Ref, 0, 3), nil},
			Accesses:   [][]Access{{{Symbol: 2, UnresolvedReason: 3, Position: 4}}, {{Location: 7, Position: 2}}},
			Extensions: [][]Extension{{{Name: 9, Value: 10}}, {{Name: 11, Value: 12}}},
		}}
}

func snapshotAttributeGraph(graph Graph) Graph {
	graph.Attributes = slices.Clone(graph.Attributes)
	graph.AttributeTables.References = cloneAttributeRows(graph.AttributeTables.References)
	graph.AttributeTables.Accesses = cloneAttributeRows(graph.AttributeTables.Accesses)
	graph.AttributeTables.Extensions = cloneAttributeRows(graph.AttributeTables.Extensions)
	return graph
}

func cloneAttributeRows[T any](rows [][]T) [][]T {
	out := slices.Clone(rows)
	for i := range out {
		out[i] = slices.Clone(out[i])
	}
	return out
}
