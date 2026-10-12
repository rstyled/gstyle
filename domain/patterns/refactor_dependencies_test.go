package patterns

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

// This is an independently specified dependency fixture, not a certification
// of Go extraction. It records one input definition, its read and return flow.
func refactorFixture(t *testing.T) (*pdg.Graph, *pdg.Builder) {
	t.Helper()
	b := pdg.NewBuilder()
	evidence, origin := b.Evidence(pdg.Supported, "Expected normal flow."), b.Origin(pdg.SourceOrigin, "Fixture source.")
	g := &pdg.Graph{Tables: b.Tables, Profile: b.Policy("fixture", "1"),
		Producer: pdg.Producer{Name: b.Text("fixture"), Version: b.Text("1"), Language: b.Text("go"), LanguageVersion: b.Known("1.26"), Configuration: b.Text("{}")},
		Sources:  []pdg.Source{{ID: b.Text("source"), Path: b.Text("source.go"), ContentIdentity: b.Known("fixture")}},
		Function: pdg.Function{ID: b.Text("f"), Name: b.Known("f"), QualifiedName: b.Known("example.f"), Span: pdg.Span{Source: 1, End: 20}, Interface: evidence,
			ReferenceCount: pdg.CountFact{Status: pdg.Known}},
		Symbols:     []pdg.Symbol{{ID: b.Text("x"), Scope: b.Text("scope"), Name: b.Known("x"), Type: b.Known("int"), Role: pdg.ParameterRole}},
		Definitions: []pdg.Definition{{ID: b.Text("definition"), Node: 1, Symbol: 1}},
		Spans:       []pdg.Span{{Source: 1, End: 20}},
	}
	for i, category := range []pdg.Category{pdg.FormalInput, pdg.Read, pdg.Return, pdg.FormalOutput} {
		g.Nodes = append(g.Nodes, pdg.Node{ID: b.Text([]string{"input", "read", "return", "output"}[i]), Category: category,
			OriginalKind: b.Known("fixture"), Span: 1, Evidence: evidence, Provenance: origin})
	}
	for _, name := range pdg.RequiredCapabilities {
		g.Capabilities = append(g.Capabilities, pdg.NamedCapability{Name: b.Text(name), Evidence: evidence})
	}
	g.Function.Inputs = []pdg.Parameter{{Name: b.Known("x"), Type: b.Known("int"), Role: pdg.ParameterRole, Symbol: 1, Variadic: pdg.BooleanFact{Status: pdg.Known}}}
	g.Nodes[0].Attributes, g.Nodes[1].Attributes = 1, 2
	if err := pdg.PackAttributes(g, []pdg.Attributes{{Defines: []pdg.Ref{1}}, {Reads: []pdg.Access{{Symbol: 1, Position: 1}}}}); err != nil {
		t.Fatal(err)
	}
	g.Edges = []pdg.Edge{
		{ID: b.Text("data"), Source: 1, Target: 2, Kind: pdg.Data, Subkind: b.Text("reaching-definition"), Symbol: 1, Definition: 1, Position: 1, Evidence: evidence, Provenance: origin},
		{ID: b.Text("value"), Source: 2, Target: 3, Kind: pdg.Data, Subkind: b.Text("value"), Evidence: evidence, Provenance: origin},
		{ID: b.Text("result"), Source: 3, Target: 4, Kind: pdg.Data, Subkind: b.Text("return"), Evidence: evidence, Provenance: origin},
		{ID: b.Text("order"), Source: 2, Target: 3, Kind: pdg.Execution, Subkind: b.Text("must-precede"), Evidence: evidence, Provenance: origin},
	}
	if err := pdg.Validate(g); err != nil {
		t.Fatal(err)
	}
	return g, b
}

func TestRefactorDependenciesRequirePreciseFacts(t *testing.T) {
	cases := []struct {
		name, gap string
		change    func(*pdg.Graph, *pdg.Builder)
	}{
		{"definition omitted", "definition-qualified", func(g *pdg.Graph, _ *pdg.Builder) { g.Edges[0].Definition = 0 }},
		{"wrong definition source", "disagrees", func(g *pdg.Graph, _ *pdg.Builder) { g.Edges[0].Source = 3 }},
		{"approximate execution", "edge evidence", func(g *pdg.Graph, b *pdg.Builder) {
			g.Edges[3].Evidence = b.Evidence(pdg.Approximated, "Partial order.")
		}},
		{"missing execution coverage", "execution_dependencies", func(g *pdg.Graph, b *pdg.Builder) { setFixtureCapability(g, b, "execution_dependencies", pdg.Unknown) }},
		{"cyclic execution", "cycle", func(g *pdg.Graph, b *pdg.Builder) {
			edge := g.Edges[3]
			edge.ID, edge.Source, edge.Target = b.Text("reverse order"), edge.Target, edge.Source
			g.Edges = append(g.Edges, edge)
		}},
		{"unknown input binding", "input has no", func(g *pdg.Graph, _ *pdg.Builder) { g.Function.Inputs[0].Symbol = 0 }},
		{"unowned write", "matching owned definition", func(g *pdg.Graph, _ *pdg.Builder) {
			g.Attributes[1].Writes = g.Attributes[1].Reads
		}},
		{"unknown call effects", "calls", func(g *pdg.Graph, b *pdg.Builder) {
			g.Nodes[1].Category = pdg.Call
			setFixtureCapability(g, b, "calls", pdg.Unknown)
		}},
		{"unknown memory", "memory_aliases", func(g *pdg.Graph, b *pdg.Builder) {
			g.Locations = []pdg.Location{{ID: b.Text("heap"), Abstraction: b.Text("unknown"), Description: b.Text("Unknown heap target."), Evidence: b.Evidence(pdg.Unknown, "Unknown alias.")}}
			setFixtureCapability(g, b, "memory_aliases", pdg.Unknown)
		}},
		{"unresolved read", "unresolved", func(g *pdg.Graph, b *pdg.Builder) {
			g.AttributeTables.Accesses[0][0].UnresolvedReason = b.Text("Unknown binding.")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			graph, builder := refactorFixture(t)
			c.change(graph, builder)
			result := CheckRefactorDependencies(graph)
			if result.Status != "partial" || !strings.Contains(strings.Join(result.Missing, " "), c.gap) {
				t.Fatalf("unexpected safety: %+v", result)
			}
		})
	}
}

func setFixtureCapability(graph *pdg.Graph, builder *pdg.Builder, name string, status pdg.Status) {
	for i, capability := range graph.Capabilities {
		if graph.Tables.Text(capability.Name) == name {
			graph.Capabilities[i].Evidence = builder.Evidence(status, "Fixture gap.")
		}
	}
}

func TestRefactorDependencyShapePreservesDirectedConstraints(t *testing.T) {
	a, _ := refactorFixture(t)
	b, builder := refactorFixture(t)
	before := append([]pdg.Edge(nil), a.Edges...)
	if result := CheckRefactorDependencies(a); result.Status != "complete" || result.ExecutionConstraints != 1 {
		t.Fatalf("expected complete fixture: %+v", result)
	}
	b.Function.ID = builder.Text("another function")
	b.Symbols[0].Scope = builder.Text("another local scope")
	b.Edges[0], b.Edges[3] = b.Edges[3], b.Edges[0]
	if !RefactorDependencyShapeMatches(a, b) {
		t.Fatal("local IDs, scope spelling or edge storage order affected the shape")
	}
	b.Edges[0].Source, b.Edges[0].Target = b.Edges[0].Target, b.Edges[0].Source
	if RefactorDependencyShapeMatches(a, b) {
		t.Fatal("reversed execution constraint accepted")
	}
	if !reflect.DeepEqual(before, a.Edges) {
		t.Fatal("borrowed canonical edges mutated")
	}
}

func TestRefactorDependenciesRejectInvalidReferences(t *testing.T) {
	graph, _ := refactorFixture(t)
	graph.Edges[0].Definition = 99
	if result := CheckRefactorDependencies(graph); result.Status != "unavailable" {
		t.Fatalf("dangling reference accepted: %+v", result)
	}
	if result := CheckRefactorDependencies(nil); result.Status != "unavailable" {
		t.Fatalf("missing graph accepted: %+v", result)
	}
}

func TestRefactorShapeKeepsBindingAndOperandIdentity(t *testing.T) {
	cases := []struct {
		name   string
		change func(*pdg.Graph, *pdg.Builder)
	}{
		{"operand position", func(g *pdg.Graph, _ *pdg.Builder) { g.Edges[0].Position = 2 }},
		{"definition ownership", func(g *pdg.Graph, _ *pdg.Builder) { g.Definitions[0].Node = 2 }},
		{"interface binding", func(g *pdg.Graph, _ *pdg.Builder) { g.Function.Inputs[0].Symbol = 0 }},
		{"binding type", func(g *pdg.Graph, b *pdg.Builder) { g.Symbols[0].Type = b.Known("bool") }},
		{"profile version", func(g *pdg.Graph, b *pdg.Builder) { g.Profile = b.Policy("fixture", "2") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, _ := refactorFixture(t)
			b, builder := refactorFixture(t)
			c.change(b, builder)
			if RefactorDependencyShapeMatches(a, b) {
				t.Fatal("different dependency identity accepted")
			}
		})
	}
}

func TestRefactorShapeDoesNotFlattenDistinctScopes(t *testing.T) {
	a, builderA := refactorFixture(t)
	b, builderB := refactorFixture(t)
	a.Symbols = append(a.Symbols, pdg.Symbol{ID: builderA.Text("local"), Scope: a.Symbols[0].Scope, Name: builderA.Known("x"), Type: builderA.Known("int"), Role: pdg.LocalRole})
	b.Symbols = append(b.Symbols, pdg.Symbol{ID: builderB.Text("local"), Scope: builderB.Text("nested scope"), Name: builderB.Known("x"), Type: builderB.Known("int"), Role: pdg.LocalRole})
	if RefactorDependencyShapeMatches(a, b) {
		t.Fatal("same spelling hid a different binding scope")
	}
}

func TestRefactorShapePreservesOpaqueSubkindAndOutcome(t *testing.T) {
	a, builderA := refactorFixture(t)
	b, builderB := refactorFixture(t)
	a.Edges[0].Subkind, a.Edges[0].Outcome = builderA.Text("a\x00b"), builderA.Text("c")
	b.Edges[0].Subkind, b.Edges[0].Outcome = builderB.Text("a"), builderB.Text("b\x00c")
	if RefactorDependencyShapeMatches(a, b) {
		t.Fatal("opaque fields lost their boundaries")
	}
}
