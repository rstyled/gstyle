package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func copyValue(t *testing.T, q *Query, target, symbol, source pdg.Ref) {
	t.Helper()
	q.representations[target] = map[pdg.Position]Representation{1: UnchangedRepresentation}
	ref := pdg.Ref(len(q.graph.Definitions) + 1)
	q.graph.Definitions = append(q.graph.Definitions, pdg.Definition{Node: target, Symbol: symbol})
	replaceValueAttrs(t, q, target, pdg.Assignment, pdg.Attributes{Writes: []pdg.Access{{Symbol: symbol, Position: 1}}, Defines: []pdg.Ref{ref}})
	if source != 0 {
		valueEdge(q, source, target, 1)
	}
}
func TestHandlingUsesCanonicalCopiesAndWrites(t *testing.T) {
	q := valueQuery(t, "read", "read", "copy")
	replaceValueAttrs(t, q, 2, pdg.Read, pdg.Attributes{Reads: []pdg.Access{{Symbol: 2, Position: 1}}})
	copyValue(t, q, 3, 2, 1)
	if !q.HandlingUnknown(3, Path{Root: 1}, Evaluation{}) {
		t.Fatal("copy of checked value becomes ignored")
	}
	q.operands[2] = nil
	valueEdge(q, 2, 3, 1)
	if q.HandlingUnknown(3, Path{Root: 1}, Evaluation{}) {
		t.Fatal("unrelated direct copy becomes handling")
	}
	q.operands[2] = nil
	if !q.HandlingUnknown(3, Path{Root: 1}, Evaluation{}) {
		t.Fatal("missing source establishes no handling")
	}
	replaceValueAttrs(t, q, 3, pdg.Assignment, pdg.Attributes{Writes: []pdg.Access{{Location: 1, Position: 1}}})
	if !q.HandlingUnknown(3, Path{Root: 1}, Evaluation{}) {
		t.Fatal("memory write becomes harmless")
	}
	replaceValueAttrs(t, q, 3, pdg.Assignment, pdg.Attributes{Writes: []pdg.Access{{Symbol: 1, Position: 1}}})
	if !q.HandlingUnknown(3, Path{Root: 1}, Evaluation{}) {
		t.Fatal("checked binding overwrite becomes harmless")
	}
}
func TestFreshConstructionNeedsAllocationIdentity(t *testing.T) {
	q := valueQuery(t, "read", "composite")
	attrs := q.graph.NodeAttributes(q.graph.Nodes[1])
	attrs.Writes = []pdg.Access{{Location: 1}}
	replaceValueAttrs(t, q, 2, pdg.Computation, attrs)
	q.allocations[1] = 2
	if q.HandlingUnknown(2, Path{Root: 1}, Evaluation{}) {
		t.Fatal("fresh construction mutates prior storage")
	}
	delete(q.allocations, 1)
	if !q.HandlingUnknown(2, Path{Root: 1}, Evaluation{}) {
		t.Fatal("unknown storage is assumed fresh")
	}
}
func TestPrintIdentityDoesNotAuthorizeShadowedCalls(t *testing.T) {
	q := valueQuery(t, "call")
	for _, tc := range []struct {
		callee  string
		form    pdg.CallForm
		unknown bool
	}{
		{"println", pdg.DirectCall, false}, {"p.println", pdg.DirectCall, true}, {"println", pdg.IndirectCall, true},
	} {
		attrs := pdg.Attributes{CallForm: tc.form, Callee: pdg.Fact{Status: pdg.Known, Value: testText(q, tc.callee)}}
		replaceValueAttrs(t, q, 1, pdg.Call, attrs)
		if got := q.HandlingUnknown(1, Path{Root: 1}, Evaluation{}); got != tc.unknown {
			t.Fatalf("%s %d unknown=%v", tc.callee, tc.form, got)
		}
	}
}
func TestStablePathUsesReachableEscapesAndReceiverCopies(t *testing.T) {
	q := valueQuery(t, "read", "&", "copy")
	valueEdge(q, 1, 2, 1)
	q.reached = []bool{true, true, true}
	if q.StablePath(Path{Root: 1}, Facts{}) {
		t.Fatal("address escape is ignored")
	}
	q.reached[1] = false
	if !q.StablePath(Path{Root: 1}, Facts{}) {
		t.Fatal("unreachable address pollutes admission")
	}
	copyValue(t, q, 3, 2, 1)
	if q.StablePath(Path{Root: 1, Fields: []uint32{0}}, Facts{}) {
		t.Fatal("receiver alias is ignored")
	}
	if !q.StablePath(Path{Root: 1}, Facts{}) {
		t.Fatal("value copy becomes receiver alias")
	}
	q.reached = nil
	if q.StablePath(Path{Root: 1}, Facts{}) {
		t.Fatal("legacy flow certifies stability")
	}
}
func TestDependencyInventoryAndShortCircuitInputs(t *testing.T) {
	q := valueQuery(t, "read", "read", "&&")
	replaceValueAttrs(t, q, 1, pdg.Read, pdg.Attributes{Reads: []pdg.Access{{Symbol: 2, Position: 1}}})
	q.required[3] = []pdg.Position{1, 2}
	valueEdge(q, 1, 3, 1)
	valueEdge(q, 2, 3, 2)
	context := Evaluation{Facts: knownValues(map[pdg.Ref]Value{1: boolValue(1)})}
	if got := q.Depends(3, Path{Root: 1}, context); got.Used || !got.Known {
		t.Fatalf("skipped RHS: %+v", got)
	}
	context.Facts.Nodes[1] = Fact{Known: true, Value: boolValue(2)}
	if got := q.Depends(3, Path{Root: 1}, context); !got.Used {
		t.Fatalf("executed RHS: %+v", got)
	}
	delete(q.operands[2], 2)
	if got := q.Depends(3, Path{Root: 1}, context); got.Known {
		t.Fatal("missing required input becomes proof of no use")
	}
}

func TestDependencyCyclesStayUnknown(t *testing.T) {
	q := valueQuery(t, "parentheses")
	q.required[1] = []pdg.Position{1}
	valueEdge(q, 1, 1, 1)
	if got := q.Depends(1, Path{Root: 2}, Evaluation{}); got.Known || got.Used {
		t.Fatalf("cycle invents absence of use: %+v", got)
	}
}
func TestMissingAddressInputCannotCertifyStability(t *testing.T) {
	q := valueQuery(t, "read", "&")
	q.reached = []bool{true, true}
	if q.StablePath(Path{Root: 1}, Facts{}) {
		t.Fatal("missing address input certifies no escape")
	}
}

func TestNonCopyMemoryEffectsStayUnknown(t *testing.T) {
	q := valueQuery(t, "opaque-effect")
	replaceValueAttrs(t, q, 1, pdg.Computation, pdg.Attributes{Writes: []pdg.Access{{Location: 1}}})
	if !q.HandlingUnknown(1, Path{Root: 1}, Evaluation{}) {
		t.Fatal("non-copy memory effect is ignored")
	}
}
func TestSharedDependencyNodesAreNotCycles(t *testing.T) {
	q := valueQuery(t, "read", "parentheses", "&&")
	q.required[2] = []pdg.Position{1}
	q.required[3] = []pdg.Position{1, 2}
	valueEdge(q, 1, 2, 1)
	valueEdge(q, 2, 3, 1)
	valueEdge(q, 2, 3, 2)
	if got := q.Depends(3, Path{Root: 2}, Evaluation{}); got.Used || !got.Known {
		t.Fatalf("shared acyclic input is not an unknown cycle: %+v", got)
	}
}
func TestAllocationIdentityBoundary(t *testing.T) {
	q := valueQuery(t, "composite")
	extension := pdg.Extension{Name: testText(q, "gstyle.memory:allocation"), Value: testText(q, `"missing"`)}
	if err := q.allocationFact(1, extension, map[string]pdg.Ref{"owner": 1}); err == nil {
		t.Fatal("unknown allocation owner admitted")
	}
	extension.Value = testText(q, `true`)
	if err := q.allocationFact(1, extension, map[string]pdg.Ref{"owner": 1}); err == nil {
		t.Fatal("malformed allocation identity admitted")
	}
	extension.Value = testText(q, `"owner"`)
	if err := q.allocationFact(1, extension, map[string]pdg.Ref{"owner": 1}); err != nil {
		t.Fatal(err)
	}
	if q.allocations[1] != 1 {
		t.Fatal("canonical allocation owner lost")
	}
}

func TestDependencyBoundariesPreserveSharedLeavesAndRootMatches(t *testing.T) {
	q := valueQuery(t, "read", "call", "pair")
	q.required[2] = []pdg.Position{1}
	q.required[3] = []pdg.Position{1, 2}
	valueEdge(q, 1, 2, 1)
	valueEdge(q, 2, 3, 1)
	valueEdge(q, 2, 3, 2)
	path := Path{Root: 1}
	if dependency := q.Depends(3, path, Evaluation{}); !dependency.Used {
		t.Fatalf("baseline dependency=%+v", dependency)
	}
	boundaries := map[pdg.Ref]bool{2: true}
	if dependency := q.DependsUntil(3, path, Evaluation{}, boundaries); dependency.Used || !dependency.Known {
		t.Fatalf("shared boundary becomes cycle or source dependency: %+v", dependency)
	}
	if len(boundaries) != 1 || !boundaries[2] {
		t.Fatal("query mutates borrowed boundaries")
	}
	if dependency := q.Depends(3, path, Evaluation{}); !dependency.Used {
		t.Fatal("boundary changes later unrestricted query")
	}
	if dependency := q.DependsUntil(1, path, Evaluation{}, map[pdg.Ref]bool{1: true}); !dependency.Used {
		t.Fatal("boundary hides its own root match")
	}
}

func TestKnownNonNilShapeRetainsPayloadDependencies(t *testing.T) {
	q := valueQuery(t, "read", "composite")
	q.required[2] = []pdg.Position{1}
	valueEdge(q, 1, 2, 1)
	context := Evaluation{Facts: knownValues(map[pdg.Ref]Value{2: {NonNil: true}})}
	if dependency := q.Depends(2, Path{Root: 1}, context); !dependency.Used || !dependency.Known {
		t.Fatalf("non-nil shape erases payload: %+v", dependency)
	}
}
