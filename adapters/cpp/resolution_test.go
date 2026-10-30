package cpp

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCPPCallTargetEvidence(t *testing.T) {
	wire := fixtureWire(t)
	var id uint64
	for i := range wire.Value.Vertices {
		node := &wire.Value.Vertices[i]
		props, err := properties(node.Properties)
		if err != nil {
			t.Fatal(err)
		}
		if node.Label == "CALL" && props["NAME"] == "cycle" {
			id, _ = identifier(node.ID)
			node.Properties["METHOD_FULL_NAME"] = json.RawMessage(`"<unresolvedNamespace>.cycle:<unresolvedSignature>(1)"`)
			break
		}
	}
	if id == 0 {
		t.Fatal("fixture lacks a recursive call")
	}
	extraction, err := readWire(t, wire)
	if err != nil {
		t.Fatal(err)
	}
	attrs := importedNode(t, extraction, id)
	if attrs.Callee.Status != pdg.Known || attrs.CallForm != pdg.DirectCall {
		t.Fatal("recorded direct target does not resolve the placeholder")
	}
	for i := len(wire.Value.Edges) - 1; i >= 0; i-- {
		edge := wire.Value.Edges[i]
		source, _ := identifier(edge.Source)
		if source == id && edge.Label == "CALL" {
			wire.Value.Edges = append(wire.Value.Edges[:i], wire.Value.Edges[i+1:]...)
		}
	}
	extraction, err = readWire(t, wire)
	if err != nil {
		t.Fatal(err)
	}
	if attrs = importedNode(t, extraction, id); attrs.Callee.Status != pdg.Unknown {
		t.Fatal("unresolved placeholder is admitted without target evidence")
	}
}

func importedNode(t *testing.T, extraction *Extraction, id uint64) pdg.Attributes {
	t.Helper()
	for _, f := range extraction.Functions {
		for _, node := range f.Graph.Nodes {
			if f.Graph.Tables.Text(node.ID) == strconv.FormatUint(id, 10) {
				return f.Graph.NodeAttributes(node)
			}
		}
	}
	t.Fatalf("node %d is not imported", id)
	return pdg.Attributes{}
}

func TestCPPAmbiguousAndIndirectCallsStayUnresolved(t *testing.T) {
	p := newProjection(&document{}, &vertex{})
	for _, targets := range [][]callTarget{
		{{Name: "a:int()"}, {Name: "b:int()"}},
		{{Name: "a:int()"}, {Name: "<unresolvedNamespace>.a:<unresolvedSignature>(0)"}},
	} {
		node := &vertex{Props: map[string]any{"DISPATCH_TYPE": "STATIC_DISPATCH", "METHOD_FULL_NAME": "a:int()"}}
		if p.callCallee(node, targets).Status != pdg.Unknown {
			t.Fatal("ambiguous or incomplete target set becomes a known callee")
		}
	}
	node := &vertex{Props: map[string]any{"DISPATCH_TYPE": "DYNAMIC_DISPATCH", "METHOD_FULL_NAME": "a:int()"}}
	if importedCallForm(node) != pdg.IndirectCall || p.callCallee(node, []callTarget{{Name: "a:int()"}}).Status != pdg.Unknown {
		t.Fatal("a dynamic candidate becomes a proven runtime target")
	}
}

func TestCPPReferencesAndCaptures(t *testing.T) {
	extraction, err := Load(context.Background(), fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, function := range extraction.Functions {
		graph := function.Graph
		name := graph.Tables.Text(graph.Function.Name.Value)
		if graph.Function.ReferenceCount.Status != pdg.Known {
			t.Fatalf("fixture %s has unknown binding count", name)
		}
		if name == "shadowed" && graph.Function.ReferenceCount.Value != 2 {
			t.Fatal("shadowed reads collapse into one binding")
		}
		if name == "closure" {
			resolved := false
			for _, node := range graph.Nodes {
				attrs := graph.NodeAttributes(node)
				if attrs.CallForm == pdg.IndirectCall && attrs.Callee.Status == pdg.Known && strings.Contains(graph.Tables.Text(attrs.Callee.Value), "<lambda>0") {
					resolved = true
				}
			}
			if !resolved {
				t.Fatal("exact lambda receiver type does not resolve its call target")
			}
		}
		if !strings.HasPrefix(name, "<lambda>") {
			continue
		}
		captureRead := false
		for _, node := range graph.Nodes {
			for _, read := range graph.NodeAttributes(node).Reads {
				if read.Symbol != 0 && graph.Symbols[read.Symbol-1].Role == pdg.CaptureRole {
					captureRead = true
				}
			}
		}
		if !captureRead {
			t.Fatal("lambda read loses its outer declaration")
		}
	}
}

func TestCPPUnresolvedBindingsPreventKnownReferenceCount(t *testing.T) {
	wire := fixtureWire(t)
	var id uint64
	for i := range wire.Value.Vertices {
		node := &wire.Value.Vertices[i]
		if node.Label == "IDENTIFIER" {
			id, _ = identifier(node.ID)
			node.Properties["NAME"] = json.RawMessage(`"unbound_identifier"`)
			node.Properties["CODE"] = json.RawMessage(`"unbound_identifier"`)
			break
		}
	}
	for i := len(wire.Value.Edges) - 1; i >= 0; i-- {
		edge := wire.Value.Edges[i]
		source, _ := identifier(edge.Source)
		if source == id && edge.Label == "REF" {
			wire.Value.Edges = append(wire.Value.Edges[:i], wire.Value.Edges[i+1:]...)
		}
	}
	extraction, err := readWire(t, wire)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, function := range extraction.Functions {
		for _, diagnostic := range function.Graph.Diagnostics {
			if function.Graph.Tables.Text(diagnostic.Code) == "cpp.unresolved-binding" {
				found = true
				if function.Graph.Function.ReferenceCount.Status != pdg.Unknown {
					t.Fatal("missing binding still claims a complete count")
				}
			}
		}
	}
	if !found {
		t.Fatal("unresolved read has no diagnostic")
	}
}

func TestCPPStableCallableRejectsEscapesWritesAndMissingDominance(t *testing.T) {
	doc, err := indexGraph(context.Background(), fixtureWire(t))
	if err != nil {
		t.Fatal(err)
	}
	var call *vertex
	for _, node := range doc.Nodes {
		if node.text("CODE") == "fn()" {
			call = node
			break
		}
	}
	if call == nil || doc.indirectTarget(call) == nil {
		t.Fatal("fixture does not prove a stable local callable")
	}
	declaration := callableDeclaration(call)
	doc.UnboundUses[call.Owner] = map[string]bool{declaration.text("NAME"): true}
	if doc.indirectTarget(call) != nil {
		t.Fatal("unbound same-name uses still prove a nonescaping callable")
	}
	delete(doc.UnboundUses, call.Owner)
	uses := doc.Referrers[declaration]
	for _, use := range []*vertex{
		{Label: "CLOSURE_BINDING"},
		{Label: "IDENTIFIER", Parent: &vertex{Label: "CALL", Props: map[string]any{"NAME": "consume"}}},
		uses[0],
	} {
		doc.Referrers[declaration] = append(append([]*vertex{}, uses...), use)
		if doc.indirectTarget(call) != nil {
			t.Fatal("escaped or reassigned callable has a proven unique target")
		}
	}
	doc.Referrers[declaration] = uses
	initializer := stableInitializer(uses)
	delete(initializer.Out, "DOMINATE")
	if doc.indirectTarget(call) != nil {
		t.Fatal("initializer without recorded dominance resolves a later call")
	}
}

func TestCPPReadBindingRecoveredFromReachingDefinitions(t *testing.T) {
	wire := fixtureWire(t)
	var use, other json.RawMessage
	for _, node := range wire.Value.Vertices {
		props, err := properties(node.Properties)
		if err != nil {
			t.Fatal(err)
		}
		if node.Label == "IDENTIFIER" && use == nil {
			use = node.ID
		}
		if node.Label == "METHOD_PARAMETER_IN" && props["NAME"] == "x" {
			other = node.ID
		}
	}
	useID, _ := identifier(use)
	for i := len(wire.Value.Edges) - 1; i >= 0; i-- {
		source, _ := identifier(wire.Value.Edges[i].Source)
		if source == useID && wire.Value.Edges[i].Label == "REF" {
			wire.Value.Edges = append(wire.Value.Edges[:i], wire.Value.Edges[i+1:]...)
		}
	}
	extraction, err := readWire(t, wire)
	if err != nil {
		t.Fatal(err)
	}
	attrs := importedNode(t, extraction, useID)
	if len(attrs.Reads) != 1 || attrs.Reads[0].Symbol == 0 {
		t.Fatal("matching reaching definition does not recover the declaration")
	}
	wire.Value.Edges = append(wire.Value.Edges, wireEdge{wireNode: wireNode{ID: json.RawMessage(`9223372036854775804`), Label: "REACHING_DEF", Properties: map[string]json.RawMessage{"VARIABLE": json.RawMessage(`"x"`)}}, Source: other, Target: use})
	extraction, err = readWire(t, wire)
	if err != nil {
		t.Fatal(err)
	}
	if attrs = importedNode(t, extraction, useID); attrs.Reads[0].Symbol != 0 || attrs.Reads[0].UnresolvedReason == 0 {
		t.Fatal("conflicting declaration identities are merged by variable spelling")
	}
}

func TestCPPRealMacroAndResolvedCallbackRecursion(t *testing.T) {
	extraction, err := Load(context.Background(), "testdata/calls.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"increment": 0, "recursive_closure": 1, "<lambda>0": 2, "dispatch_unknown": 0}
	macros, resolved, unresolved := 0, 0, 0
	for _, function := range extraction.Functions {
		graph := function.Graph
		name := graph.Tables.Text(graph.Function.Name.Value)
		if score, ok := want[name]; !ok || score != function.Score.CognitiveComplexity {
			t.Fatalf("%s: cognitive score %d, want %d", name, function.Score.CognitiveComplexity, score)
		}
		delete(want, name)
		for _, node := range graph.Nodes {
			attrs := graph.NodeAttributes(node)
			if graph.Tables.Text(attrs.Construct) == "macro-expansion" {
				macros++
				if node.Category == pdg.Call {
					t.Fatal("real macro wrapper becomes a runtime call")
				}
			}
			if node.Category != pdg.Call {
				continue
			}
			if attrs.Callee.Status != pdg.Known {
				unresolved++
			} else if attrs.CallForm == pdg.IndirectCall {
				resolved++
			}
		}
	}
	if len(want) != 0 || macros != 1 || resolved != 1 || unresolved != 1 {
		t.Fatalf("wrong real export coverage: missing=%v macros=%d resolved=%d unresolved=%d", want, macros, resolved, unresolved)
	}
}

func TestCPPInlinedMacroIsNotARuntimeCall(t *testing.T) {
	doc, err := indexGraph(context.Background(), fixtureWire(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range doc.Nodes {
		if node.Label == "CALL" && node.text("NAME") == "cycle" {
			node.Props["DISPATCH_TYPE"] = "INLINED"
			extraction, err := convert(context.Background(), doc)
			if err != nil {
				t.Fatal(err)
			}
			attrs := importedNode(t, extraction, node.ID)
			if category(node) != pdg.Computation || attrs.CallForm != pdg.UnrecordedCall || attrs.Callee.Value != 0 {
				t.Fatal("macro expansion is reported as an unknown runtime call")
			}
			return
		}
	}
	t.Fatal("fixture has no call to classify")
}
