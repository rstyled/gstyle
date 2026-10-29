package joern

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	want := map[string][2]int{"straight": {1, 0}, "nested": {3, 3}, "alternatives": {3, 3}, "logical": {3, 1}, "mixed": {4, 3}, "selection": {3, 1}, "cycle": {2, 2}, "mutual_a": {2, 2}, "mutual_b": {2, 2}, "chain": {1, 0}, "caught": {3, 2}, "conditional": {2, 1}, "looping": {3, 3}, "<lambda>0": {2, 1}, "closure": {1, 0}}
	if len(extraction.Functions) != len(want) {
		t.Fatalf("got %d methods, want %d", len(extraction.Functions), len(want))
	}
	for _, fn := range extraction.Functions {
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
	if report.Quality.Status != "unavailable" || report.Functions != len(want) {
		t.Fatalf("misleading report: %+v", report)
	}
	if len(extraction.Patterns().Candidates) == 0 {
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
