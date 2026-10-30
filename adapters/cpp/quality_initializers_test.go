package cpp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func zoneDocument(t *testing.T) *document {
	t.Helper()
	raw, err := os.ReadFile("testdata/zone.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	var wire wireGraph
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	doc, err := indexGraph(context.Background(), wire)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestCPPInitializationsDoNotUseMutationBudgets(t *testing.T) {
	extracted, err := Load(context.Background(), "testdata/zone.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(extracted.Functions) != 1 {
		t.Fatal("the reduced RocketRide export loses its method")
	}
	function := extracted.Functions[0]
	fact := function.Quality
	if len(fact.Mutations) != 0 || len(fact.Calls) != 7 {
		t.Fatalf("local initializations count as mutations or hide calls: %+v", fact)
	}
	writes := 0
	for _, node := range function.Graph.Nodes {
		if node.Category == pdg.Assignment {
			writes += len(function.Graph.NodeAttributes(node).Writes)
		}
	}
	if writes != 4 {
		t.Fatal("initial definitions disappear from the canonical PDG")
	}
	for _, diagnostic := range extracted.Report().Quality.Report.Diagnostics {
		if diagnostic.RuleID == "mutated_targets" || diagnostic.RuleID == "mutation_per_target" {
			t.Fatalf("initialization creates a mutation finding: %+v", diagnostic)
		}
	}
	lifetimes, returns := 0, 0
	for _, effect := range fact.Effects {
		if effect.Line <= 30 || effect.Line > 36 {
			t.Fatalf("metadata creates evidence on the wrong source line: %+v", effect)
		}
		if strings.HasPrefix(effect.Detail, "unresolved C++ object lifetime:") {
			lifetimes++
		}
		if effect.Detail == "unresolved C++ return copy or move" && effect.Line == 36 {
			returns++
		}
	}
	if lifetimes != 4 || returns != 1 {
		t.Fatalf("object lifetime uncertainty is lost or duplicated: lifetimes=%d returns=%d", lifetimes, returns)
	}
}

func TestCPPInitializerRequiresBoundDeclaratorRange(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*vertex, *vertex)
	}{
		{"later assignment on the same line", func(n, _ *vertex) { n.Props["OFFSET"] = json.Number("1517") }},
		{"different initializer end", func(n, _ *vertex) { n.Props["OFFSET_END"] = json.Number("1583") }},
		{"missing offsets", func(n, _ *vertex) { delete(n.Props, "OFFSET") }},
		{"compound assignment", func(n, _ *vertex) { n.Props["NAME"] = "<operator>.assignmentPlus" }},
		{"unbound target", func(n, _ *vertex) { delete(argument(n, 1).Out, "REF") }},
		{"indirect target", func(n, _ *vertex) { argument(n, 1).Label = "CALL" }},
		{"another scope", func(_, local *vertex) { local.Parent = nil }},
		{"static local", func(_, local *vertex) { local.Props["CODE"] = "static auto now" }},
		{"captured storage", func(_, local *vertex) { local.Props["CLOSURE_BINDING_ID"] = "capture" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			doc := zoneDocument(t)
			for _, node := range doc.Nodes {
				if node.ID != 30064808131 {
					continue
				}
				local := assignedLocal(node)
				if !localInitializer(node) || local == nil {
					t.Fatal("recorded declaration is not an initializer")
				}
				test.change(node, local)
				if localInitializer(node) {
					t.Fatal("a write without initialization evidence is dropped")
				}
				return
			}
			t.Fatal("fixture initializer missing")
		})
	}
}
