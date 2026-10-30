package cpp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

const fixture = "testdata/sample.graphson.json"

func fixtureWire(t *testing.T) wireGraph {
	t.Helper()
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var wire wireGraph
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	return wire
}

func readWire(t *testing.T, wire wireGraph) (*Extraction, error) {
	t.Helper()
	data, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	return Read(context.Background(), bytes.NewReader(data))
}

func TestRealCPPExportComplexity(t *testing.T) {
	extraction, err := Load(context.Background(), fixture)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]int{"straight": {1, 0}, "nested": {3, 3}, "alternatives": {3, 3}, "logical": {3, 1}, "mixed": {4, 3}, "selection": {3, 1}, "cycle": {2, 2}, "mutual_a": {2, 2}, "mutual_b": {2, 2}, "chain": {1, 0}, "caught": {3, 2}, "conditional": {2, 1}, "looping": {3, 3}, "<lambda>0": {2, 1}, "closure": {1, 0}, "shadowed": {1, 0}, "braced": {3, 4}, "negated": {3, 2}, "predicate": {3, 2}, "touch": {2, 1}}
	if len(extraction.Functions) != len(want) {
		t.Fatalf("got %d methods, want %d", len(extraction.Functions), len(want))
	}
	for _, fn := range extraction.Functions {
		if fn.Score.Package != "cpp" {
			t.Errorf("method package = %q, want cpp", fn.Score.Package)
		}
		name := fn.Graph.Tables.Text(fn.Graph.Function.Name.Value)
		scores, exists := want[name]
		if !exists || scores != [2]int{fn.Score.Complexity, fn.Score.CognitiveComplexity} {
			t.Errorf("%s: CC=%d COG=%d, want %v", name, fn.Score.Complexity, fn.Score.CognitiveComplexity, scores)
		}
		if fn.Score.Source == "" || fn.Graph.Function.Span.End <= fn.Graph.Function.Span.Start {
			t.Errorf("%s: missing source snapshot", name)
		}
		if len(fn.Score.CyclomaticDiagnostics) != fn.Score.Complexity {
			t.Errorf("%s: score lacks diagnostic witnesses", name)
		}
		if err := pdg.Validate(fn.Graph); err != nil {
			t.Fatal(err)
		}
	}
	report := extraction.Report()
	if report.Quality.Status != "ready" || report.Quality.Report.Summary.Functions != len(want) || report.Functions != len(want) {
		t.Fatalf("misleading report: %+v", report)
	}
	if len(extraction.Patterns().Measurements) == 0 {
		t.Fatal("imported dependencies do not reach the slicing consumer")
	}
}

func TestRealCPPDependencies(t *testing.T) {
	wire := fixtureWire(t)
	doc, err := indexGraph(context.Background(), wire)
	if err != nil {
		t.Fatal(err)
	}
	extraction, err := convert(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, link := range doc.Edges {
		if link.Source.Owner == link.Target.Owner && link.Source.Owner != nil && sourceMethod(link.Source.Owner) {
			counts[link.Label]++
		}
	}
	got := map[string]int{}
	for _, fn := range extraction.Functions {
		graph := fn.Graph
		if graph.Tables.Text(graph.Tables.Policies[graph.Profile-1].ID) != "gstyle.cpp.import" || graph.Tables.Text(graph.Producer.Name) != "gstyle-cpp" {
			t.Fatal("unexpected C++ graph identity")
		}
		if graph.Conformance.Status != pdg.NotEvaluated || graph.NormalFlow != nil {
			t.Fatal("import grants native source conformance")
		}
		for _, link := range graph.Edges {
			if link.Kind == pdg.Execution {
				t.Fatal("CFG successor became must-precede dependency")
			}
			got[graph.Tables.Text(link.Subkind)]++
		}
		for _, input := range graph.Function.Inputs {
			if input.Symbol == 0 {
				t.Fatal("parameter loses declaration identity")
			}
		}
		for _, node := range graph.Nodes {
			attrs := graph.NodeAttributes(node)
			if strings.HasPrefix(graph.Tables.Text(attrs.Operator.Value), "<operator>.") && node.Category == pdg.Call {
				t.Fatal("operator classified as an effectful call")
			}
		}
	}
	for _, kind := range []string{"CDG", "REACHING_DEF"} {
		if got[kind] == 0 || got[kind] != counts[kind] {
			t.Errorf("%s: got %d edges, want %d", kind, got[kind], counts[kind])
		}
	}
	if got["value-operand"] == 0 {
		t.Fatal("expression operands do not feed consumers")
	}
}

func TestRejectMalformedGraph(t *testing.T) {
	cases := map[string]func(*wireGraph){
		"duplicate vertex":  func(w *wireGraph) { w.Value.Vertices = append(w.Value.Vertices, w.Value.Vertices[0]) },
		"duplicate edge":    func(w *wireGraph) { w.Value.Edges = append(w.Value.Edges, w.Value.Edges[0]) },
		"dangling endpoint": func(w *wireGraph) { w.Value.Edges[0].Target = json.RawMessage(`9223372036854775806`) },
		"fractional ID":     func(w *wireGraph) { w.Value.Vertices[0].ID = json.RawMessage(`1.5`) },
		"negative ID":       func(w *wireGraph) { w.Value.Vertices[0].ID = json.RawMessage(`-1`) },
		"invalid scalar": func(w *wireGraph) {
			w.Value.Vertices[0].Properties["NAME"] = json.RawMessage(`{"@value":["one","two"]}`)
		},
		"fractional line": func(w *wireGraph) { w.Value.Vertices[0].Properties["LINE_NUMBER"] = json.RawMessage(`1.25`) },
		"multiple AST parents": func(w *wireGraph) {
			for _, e := range w.Value.Edges {
				if e.Label == "AST" {
					e.ID = json.RawMessage(`9223372036854775805`)
					w.Value.Edges = append(w.Value.Edges, e)
					break
				}
			}
		},
		"missing metadata": func(w *wireGraph) {
			for i := range w.Value.Vertices {
				if w.Value.Vertices[i].Label == "META_DATA" {
					w.Value.Vertices[i].Label = "UNKNOWN"
				}
			}
		},
		"foreign language": func(w *wireGraph) {
			for i := range w.Value.Vertices {
				if w.Value.Vertices[i].Label == "META_DATA" {
					w.Value.Vertices[i].Properties["LANGUAGE"] = json.RawMessage(`{"@value":"JAVA"}`)
				}
			}
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			wire := fixtureWire(t)
			change(&wire)
			if _, err := readWire(t, wire); err == nil {
				t.Fatal("invalid graph accepted")
			}
		})
	}
}

func TestInputErrorsAndCancellation(t *testing.T) {
	for _, input := range []string{"{}", "null", "[]", `{"@type":"tinker:graph","@value":{}}`} {
		if _, err := Read(context.Background(), strings.NewReader(input)); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	data, _ := os.ReadFile(fixture)
	if _, err := Read(context.Background(), bytes.NewReader(append(data, []byte("{}")...))); err == nil {
		t.Fatal("accepted trailing graph")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Read(ctx, bytes.NewReader(data)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := Extract(context.Background(), nil); err == nil {
		t.Fatal("missing input accepted")
	}
}

func TestSourceAbsentStaysUnavailable(t *testing.T) {
	wire := fixtureWire(t)
	for i := range wire.Value.Vertices {
		if wire.Value.Vertices[i].Label == "FILE" {
			delete(wire.Value.Vertices[i].Properties, "CONTENT")
		}
	}
	extraction, err := readWire(t, wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range extraction.Functions {
		if fn.Graph.Sources[0].Snapshot != 0 || fn.Graph.Sources[0].ContentIdentity.Status != pdg.Unknown {
			t.Fatal("absent source has verified identity")
		}
		for _, node := range fn.Graph.Nodes {
			if node.Span != 0 {
				t.Fatal("absent source has invented byte span")
			}
		}
	}
}

func TestExactVertexIDs(t *testing.T) {
	for _, id := range []uint64{0, 9007199254740993} {
		wire := fixtureWire(t)
		old := uint64(0)
		for i := range wire.Value.Vertices {
			if wire.Value.Vertices[i].Label == "METHOD" {
				old, _ = identifier(wire.Value.Vertices[i].ID)
				wire.Value.Vertices[i].ID = json.RawMessage(fmt.Sprint(id))
				break
			}
		}
		for i := range wire.Value.Edges {
			source, _ := identifier(wire.Value.Edges[i].Source)
			target, _ := identifier(wire.Value.Edges[i].Target)
			if source == old {
				wire.Value.Edges[i].Source = json.RawMessage(fmt.Sprint(id))
			}
			if target == old {
				wire.Value.Edges[i].Target = json.RawMessage(fmt.Sprint(id))
			}
		}
		extraction, err := readWire(t, wire)
		if err != nil {
			t.Fatal(err)
		}
		if extraction.Functions[0].Graph.Tables.Text(extraction.Functions[0].Graph.Function.ID) != fmt.Sprintf("cpp:%d", id) {
			t.Fatal("vertex ID is rounded or treated as absent")
		}
	}
	if _, err := identifier(json.RawMessage(`{"@type":"g:Double","@value":1}`)); err == nil {
		t.Fatal("floating ID wrapper accepted")
	}
}

func TestRejectASTCycleAndMissingOperand(t *testing.T) {
	wire := fixtureWire(t)
	first := wireNode{ID: json.RawMessage(`9223372036854775800`), Label: "BLOCK"}
	second := wireNode{ID: json.RawMessage(`9223372036854775801`), Label: "BLOCK"}
	wire.Value.Vertices = append(wire.Value.Vertices, first, second)
	wire.Value.Edges = append(wire.Value.Edges,
		wireEdge{wireNode: wireNode{ID: json.RawMessage(`9223372036854775802`), Label: "AST"}, Source: first.ID, Target: second.ID},
		wireEdge{wireNode: wireNode{ID: json.RawMessage(`9223372036854775803`), Label: "AST"}, Source: second.ID, Target: first.ID})
	if _, err := readWire(t, wire); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("AST cycle: %v", err)
	}
	wire = fixtureWire(t)
	var logicalID uint64
	for _, node := range wire.Value.Vertices {
		props, err := properties(node.Properties)
		if err != nil {
			t.Fatal(err)
		}
		if props["NAME"] == "<operator>.logicalAnd" {
			logicalID, _ = identifier(node.ID)
			break
		}
	}
	for i, link := range wire.Value.Edges {
		source, _ := identifier(link.Source)
		if source == logicalID && link.Label == "AST" {
			wire.Value.Edges = append(wire.Value.Edges[:i], wire.Value.Edges[i+1:]...)
			break
		}
	}
	if _, err := readWire(t, wire); err == nil {
		t.Fatal("logical operator without two operands accepted")
	}
}

func TestShadowBindingsAndOperandDirection(t *testing.T) {
	extraction, err := Load(context.Background(), fixture)
	if err != nil {
		t.Fatal(err)
	}
	var returnOperand bool
	for _, fn := range extraction.Functions {
		g := fn.Graph
		if g.Tables.Text(g.Function.Name.Value) == "shadowed" {
			var identities []pdg.Text
			for _, symbol := range g.Symbols {
				if g.Tables.Text(symbol.Name.Value) == "value" {
					identities = append(identities, symbol.ID)
				}
			}
			if len(identities) != 2 || identities[0] == identities[1] {
				t.Fatal("shadowed bindings are merged by spelling")
			}
		}
		for _, link := range g.Edges {
			if g.Tables.Text(link.Subkind) != "value-operand" {
				continue
			}
			source, target := g.Nodes[link.Source-1], g.Nodes[link.Target-1]
			if source.Category == pdg.Read && target.Category == pdg.Return {
				returnOperand = true
			}
			attrs := g.NodeAttributes(target)
			if g.Tables.Text(attrs.Operator.Value) == "<operator>.assignment" && link.Position == 2 {
				t.Fatal("assignment destination is read as a value")
			}
		}
	}
	if !returnOperand {
		t.Fatal("return does not consume its expression")
	}
}

func TestIncomingDependenciesSurviveExcludedOwner(t *testing.T) {
	doc, err := indexGraph(context.Background(), fixtureWire(t))
	if err != nil {
		t.Fatal(err)
	}
	var source *vertex
	for _, node := range doc.Nodes {
		if node.Label == "METHOD" && node.text("NAME") == "<global>" {
			source = node
			break
		}
	}
	if source == nil {
		t.Fatal("fixture has no global owner")
	}
	target := doc.Methods[0]
	const importedID = uint64(9223372036854775806)
	doc.Edges = append(doc.Edges, edge{ID: importedID, Source: source, Target: target, Label: "REACHING_DEF", Variable: "global"})
	doc.indexSources()
	extraction, err := convert(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	graph := extraction.Functions[0].Graph
	found := false
	for _, extension := range graph.Extensions {
		if graph.Tables.Text(extension.Name) != "gstyle.cpp:cross-method-dependencies" {
			continue
		}
		var links []struct {
			ID     uint64 `json:"id"`
			Source uint64 `json:"source"`
			Target uint64 `json:"target"`
		}
		if err := json.Unmarshal([]byte(graph.Tables.Text(extension.Value)), &links); err != nil {
			t.Fatal(err)
		}
		for _, link := range links {
			if link.ID == importedID && link.Source == source.ID && link.Target == target.ID {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("incoming dependency from excluded global method was dropped")
	}
}

func TestVoidMethodInterface(t *testing.T) {
	extraction, err := Load(context.Background(), fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range extraction.Functions {
		g := fn.Graph
		if g.Tables.Text(g.Function.Name.Value) != "touch" {
			continue
		}
		if len(g.Function.Outputs) != 0 {
			t.Fatal("void method has a value-output slot")
		}
		address := false
		for _, link := range g.Edges {
			if g.Tables.Text(link.Subkind) == "write-address" {
				address = true
			}
		}
		if !address {
			t.Fatal("pointer destination loses its write-address dependency")
		}
		for _, node := range g.Nodes {
			if g.Tables.Text(node.OriginalKind.Value) == "METHOD_RETURN" && node.Category == pdg.Exit {
				return
			}
		}
		t.Fatal("void method lacks an exit node")
	}
	t.Fatal("void source method is not imported")
}
