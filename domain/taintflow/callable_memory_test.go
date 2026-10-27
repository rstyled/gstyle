package taintflow

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func callableMemoryFixture(t testing.TB, target string) evaluation {
	t.Helper()
	q := projectionQuery(t, "caller", 0)
	graph := q.graph
	text := func(value string) pdg.Text {
		graph.Tables.Strings = append(graph.Tables.Strings, value)
		return pdg.Text(len(graph.Tables.Strings))
	}
	graph.Attributes[graph.Nodes[2].Attributes-1].Callee = pdg.Fact{Status: pdg.Known, Value: text(target)}
	for index, kind := range []string{`"interface"`, `"function"`} {
		graph.AttributeTables.Extensions = append(graph.AttributeTables.Extensions, []pdg.Extension{{Name: text("gstyle.memory:kind"), Value: text(kind)}})
		graph.Attributes = append(graph.Attributes, pdg.AttributeSet{Extensions: pdg.Ref(len(graph.AttributeTables.Extensions))})
		graph.Nodes[index].Attributes = pdg.Ref(len(graph.Attributes))
	}
	return evaluation{query: q, state: emptyState(), catalog: map[string]*query{
		"pure": {memoryPreserved: true}, "otherPure": {memoryPreserved: true}, "writer": {},
	}}
}

func callableMemoryValue(names ...string) value {
	item := emptyValue()
	for _, name := range names {
		item.Functions[name] = true
	}
	return item
}

func TestCallableMemoryProofRequiresEveryBody(t *testing.T) {
	for _, tc := range []struct {
		name   string
		names  []string
		opaque bool
		want   bool
	}{
		{"empty", nil, false, false},
		{"pure", []string{"pure"}, false, true},
		{"all pure", []string{"pure", "otherPure"}, false, true},
		{"writer", []string{"writer"}, false, false},
		{"mixed", []string{"pure", "writer"}, false, false},
		{"missing body", []string{"pure", "missing"}, false, false},
		{"opaque", []string{"pure"}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := callableMemoryFixture(t, "sort.Slice")
			item := callableMemoryValue(tc.names...)
			item.Opaque = tc.opaque
			if got := e.callablePreservesMemory(item); got != tc.want {
				t.Fatalf("memory preserved=%v want %v", got, tc.want)
			}
		})
	}
}

func TestSortHavocRetainsComparatorReads(t *testing.T) {
	for _, host := range []string{"sort.Slice", "sort.SliceStable"} {
		for _, tc := range []struct {
			name   string
			names  []string
			opaque bool
			havoc  bool
		}{
			{"pure", []string{"pure"}, false, false},
			{"all pure", []string{"pure", "otherPure"}, false, false},
			{"writer", []string{"writer"}, false, true},
			{"mixed", []string{"pure", "writer"}, false, true},
			{"missing body", []string{"missing"}, false, true},
			{"unknown", nil, true, true},
			{"opaque known", []string{"pure"}, true, true},
		} {
			t.Run(host+"/"+tc.name, func(t *testing.T) {
				e := callableMemoryFixture(t, host)
				first := projectionValue("slice", object{Input: 1})
				captured := object{Captured: true, Allocation: "comparator cell"}
				comparator := callableMemoryValue(tc.names...)
				comparator.Objects[captured] = true
				comparator.Opaque = tc.opaque
				e.state.Nodes[slot{Node: 1}] = first
				e.state.Nodes[slot{Node: 2}] = comparator
				payload := projectionValue("capture taint")
				e.state.Memory[cellKey(captured, "value")] = payload
				read := e.unknownCall(3)
				if !setContains(read.Data, payload.Data) {
					t.Fatal("comparator capture read lost taint")
				}
				if _, got := e.state.Memory[cellKey(captured, "*")]; got != tc.havoc {
					t.Fatalf("comparator havoc=%v want %v", got, tc.havoc)
				}
				if _, got := e.state.Memory[cellKey(object{Input: 1}, callbackEffect)]; !got {
					t.Fatal("slice interface argument loses pending writes")
				}
			})
		}
	}
}

func TestSortHavocKeepsAliasesOutsideComparator(t *testing.T) {
	for _, host := range []string{"sort.Slice", "sort.SliceStable"} {
		for _, alias := range []string{"slice interface", "slice known function", "target known function"} {
			t.Run(host+"/"+alias, func(t *testing.T) {
				e := callableMemoryFixture(t, host)
				captured := object{Captured: true, Allocation: "shared cell"}
				comparator := callableMemoryValue("pure")
				comparator.Objects[captured] = true
				e.state.Nodes[slot{Node: 2}] = comparator
				other := comparator
				position := pdg.Ref(1)
				if alias == "slice interface" {
					other.Functions = map[string]bool{}
				}
				if alias == "target known function" {
					position = 0
				}
				e.state.Nodes[slot{Node: position}] = other
				e.havocCallableInputs(3, projectionValue("incoming"))
				if _, found := e.state.Memory[cellKey(captured, "*")]; !found {
					t.Fatal("alias outside comparator loses write path")
				}
			})
		}
	}
}

func TestSortMemoryContractRequiresResolvedDirectIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target string
		form   pdg.CallForm
		status pdg.Status
		want   bool
	}{
		{"slice", "sort.Slice", pdg.DirectCall, pdg.Known, true},
		{"stable", "sort.SliceStable", pdg.DirectCall, pdg.Known, true},
		{"unknown identity", "sort.Slice", pdg.DirectCall, pdg.Unknown, false},
		{"unsupported identity", "sort.Slice", pdg.DirectCall, pdg.Unsupported, false},
		{"indirect", "sort.Slice", pdg.IndirectCall, pdg.Known, false},
		{"unknown form", "sort.Slice", pdg.UnknownCall, pdg.Known, false},
		{"lookalike", "example.sort.Slice", pdg.DirectCall, pdg.Known, false},
		{"other sort API", "sort.SliceIsSorted", pdg.DirectCall, pdg.Known, false},
		{"other host", "example.host", pdg.DirectCall, pdg.Known, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := callableMemoryFixture(t, tc.target)
			attrs := &e.query.graph.Attributes[e.query.graph.Nodes[2].Attributes-1]
			attrs.CallForm, attrs.Callee.Status = tc.form, tc.status
			comparator := callableMemoryValue("pure")
			captured := object{Captured: true, Allocation: "cell"}
			comparator.Objects[captured] = true
			e.state.Nodes[slot{Node: 2}] = comparator
			if got := e.sortsSlice(3); got != tc.want {
				t.Fatalf("sort contract=%v want %v", got, tc.want)
			}
			if got := e.mutableCallableReferences(3).Objects[captured]; got == tc.want {
				t.Fatalf("comparator kept=%v want %v", got, !tc.want)
			}
		})
	}
}
