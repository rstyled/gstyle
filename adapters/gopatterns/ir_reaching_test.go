package gopatterns

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Expectations identify source operations, not construction-order IDs. They
// independently specify which writes can reach each read in these Go examples.
func TestScalarReachingDefinitionFixtures(t *testing.T) {
	cases := []struct {
		name, function string
		want           []string
	}{
		{"overwrite", `func f(x int) int { x = 1; x = 2; return x }`, []string{"x = 2 -> return x : x"}},
		{"rhs-before-write", `func f(x int) int { x = x + 1; return x }`, []string{"x int -> x + 1 : x", "x = x + 1 -> return x : x"}},
		{"branch-join", `func f(x int, c bool) int { if c { x = 1 }; return x }`, []string{"c bool -> if c { x = 1 } : c", "x int -> return x : x", "x = 1 -> return x : x"}},
		{"both-arms", `func f(x int, c bool) int { if c { x = 1 } else { x = 2 }; return x }`, []string{"c bool -> if c { x = 1 } else { x = 2 } : c", "x = 1 -> return x : x", "x = 2 -> return x : x"}},
		{"branch-overwrite", `func f(x int, c bool) int { if c { x = 1 }; x = 2; return x }`, []string{"c bool -> if c { x = 1 } : c", "x = 2 -> return x : x"}},
		{"early-return", `func f(x int, c bool) int { if c { x = 1; return x }; return x }`, []string{"c bool -> if c { x = 1; return x } : c", "x = 1 -> return x #1 : x", "x int -> return x #2 : x"}},
		{"zero-value", `func f() int { var x int; return x }`, []string{"x int -> return x : x"}},
		{"parallel-assignment", `func f(x int, y int) int { x, y = y, x; return x }`, []string{"x int -> x, y = y, x : x", "y int -> x, y = y, x : y", "x, y = y, x -> return x : x"}},
		{"compound", `func f(x int) int { x += 1; return x }`, []string{"x int -> x += 1 : x", "x += 1 -> return x : x"}},
		{"increment", `func f(x int) int { x++; return x }`, []string{"x int -> x++ : x", "x++ -> return x : x"}},
		{"repeated-read", `func f(x int) int { return x + x }`, []string{"x int -> x + x : x", "x int -> x + x : x"}},
		{"shadow", `func f(x int, c bool) int { if c { x := 2; return x }; return x }`, []string{"c bool -> if c { x := 2; return x } : c", "x := 2 -> return x #1 : x", "x int -> return x #2 : x"}},
		{"unreachable", `func f(x int) int { return x; x = 2; return x }`, []string{"x int -> return x #1 : x"}},
		{"named-result", `func f() (x int) { x = 2; return }`, []string{"x = 2 -> return : x"}},
		{"same-text-arms", `func f(c bool) int { var x int; if c { x = 1 } else { x = 1 }; return x }`, []string{"c bool -> if c { x = 1 } else { x = 1 } : c", "x = 1 #1 -> return x : x", "x = 1 #2 -> return x : x"}},
		{"predicate-read", `func f(x int) int { if x > 0 { x = 1 }; return x }`, []string{"x int -> x > 0 : x", "x int -> return x : x", "x = 1 -> return x : x"}},
		{"nested-join", `func f(x int, a bool, b bool) int { if a { if b { x = 1 } else { x = 2 } }; return x }`, []string{"a bool -> if a { if b { x = 1 } else { x = 2 } } : a", "b bool -> if b { x = 1 } else { x = 2 } : b", "x int -> return x : x", "x = 1 -> return x : x", "x = 2 -> return x : x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package example\n" + tc.function
			g := parseIR(t, source, pdg.NewBuilder())
			again := parseIR(t, source, pdg.NewBuilder())
			if !reflect.DeepEqual(g, again) {
				t.Fatal("identical source produces different IR across runs")
			}
			if err := pdg.Validate(&g); err != nil {
				t.Fatal(err)
			}
			for _, edge := range g.Edges {
				if g.Tables.Text(edge.Subkind) == "mining-value" {
					t.Fatal("stale first-binding edge retained")
				}
			}
			got := reachingDescriptions(t, g, source)
			slices.Sort(tc.want)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v; want %v", got, tc.want)
			}
			if reachingCapability(g) != pdg.Supported {
				t.Fatal("supported subset lacks reaching-definition evidence")
			}
			if g.Conformance.Status != pdg.NonConformant {
				t.Fatal("one completed capability is not extraction conformance")
			}
		})
	}
}

func reachingDescriptions(t *testing.T, g pdg.Graph, source string) []string {
	t.Helper()
	var descriptions []string
	for _, edge := range g.Edges {
		if g.Tables.Text(edge.Subkind) != "reaching-definition" {
			continue
		}
		if g.Nodes[edge.Target-1].Category == pdg.Merge {
			continue
		}
		definition := g.Definitions[edge.Definition-1]
		if edge.Source != definition.Node || edge.Symbol != definition.Symbol {
			t.Fatal("edge disagrees with definition")
		}
		if g.Tables.Evidence[edge.Evidence-1].Status != pdg.Supported {
			t.Fatal("exact edge lacks support evidence")
		}
		from, to := g.Spans[g.Nodes[edge.Source-1].Span-1], g.Spans[g.Nodes[readConsumer(g, edge.Target)-1].Span-1]
		name := g.Tables.Text(g.Symbols[edge.Symbol-1].Name.Value)
		for _, producer := range reachingProducers(g, edge.Source) {
			from = g.Spans[g.Nodes[producer-1].Span-1]
			descriptions = append(descriptions, sourceOperation(source, from)+" -> "+sourceOperation(source, to)+" : "+name)
		}
	}
	slices.Sort(descriptions)
	return descriptions
}

func sourceOperation(source string, span pdg.Span) string {
	text := strings.TrimSpace(source[span.Start:span.End])
	if strings.Count(source, text) <= 1 {
		return text
	}
	return text + " #" + strconv.Itoa(strings.Count(source[:span.Start], text)+1)
}

func reachingCapability(g pdg.Graph) pdg.Status {
	for _, capability := range g.Capabilities {
		if g.Tables.Text(capability.Name) == "scalar_reaching_definitions" {
			return g.Tables.Evidence[capability.Evidence-1].Status
		}
	}
	return pdg.Unknown
}

func TestReachingDefinitionsRejectUnsupportedFunctions(t *testing.T) {
	functions := []string{
		`func f(x int) int { for x < 2 { println(x); x++ }; return x }`,
		`func f(p *int) int { return *p }`,
		`func f(x int) int { println(x); return x }`,
		`func f(xs []int) int { return xs[0] }`,
		`var x int; func f() int { x = 1; return x }`,
		`func f(x int) int { next := func() int { return x }; return next() }`,
	}
	for _, function := range functions {
		g := parseIR(t, "package example\n"+function, pdg.NewBuilder())
		if reachingCapability(g) == pdg.Supported {
			t.Fatalf("unsupported function claims support: %s", function)
		}
		for _, edge := range g.Edges {
			if edge.Definition != 0 && g.Tables.Evidence[edge.Evidence-1].Status == pdg.Supported {
				t.Fatalf("partial reaching definitions emitted: %s", function)
			}
		}
	}
}

func TestScalarValueEdgesPreserveExpressionFlow(t *testing.T) {
	source := "package example\nfunc f(x int) int { x = (x + 1); return x * 2 }"
	g := parseIR(t, source, pdg.NewBuilder())
	var got []string
	for _, edge := range g.Edges {
		if g.Tables.Text(edge.Subkind) != "scalar-value" {
			continue
		}
		from, to := g.Spans[g.Nodes[edge.Source-1].Span-1], g.Spans[g.Nodes[edge.Target-1].Span-1]
		got = append(got, sourceOperation(source, from)+" -> "+sourceOperation(source, to))
	}
	want := []string{"1 -> x + 1", "x + 1 -> (x + 1)", "(x + 1) -> x = (x + 1)", "2 -> x * 2", "x * 2 -> return x * 2", "x #4 -> x + 1", "x #5 -> x * 2"}
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v; want %v", got, want)
	}
}

func TestReachingEdgesKeepOperandOrder(t *testing.T) {
	cases := []struct {
		function string
		want     []string
	}{
		{`func f(x int, y int) int { return y - x }`, []string{"y:1", "x:2"}},
		{`func f(x int, y int) { x, y = y, x }`, []string{"y:1", "x:2"}},
		{`func f(x int, y int) { x += y }`, []string{"x:1", "y:2"}},
		{`func f(x int) int { return x + x }`, []string{"x:1", "x:2"}},
	}
	for _, tc := range cases {
		g := parseIR(t, "package example\n"+tc.function, pdg.NewBuilder())
		var got []string
		for _, edge := range g.Edges {
			if edge.Definition == 0 {
				continue
			}
			got = append(got, g.Tables.Text(g.Symbols[edge.Symbol-1].Name.Value)+":"+strconv.Itoa(int(readOperandPosition(g, edge))))
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %v; want %v", tc.function, got, tc.want)
		}
	}
}

// Resolve explicit value joins for the existing source def/use expectations.
// Separate join fixtures assert the actual intermediate graph entities.
func reachingProducers(g pdg.Graph, node pdg.Ref) []pdg.Ref {
	if g.Nodes[node-1].Category != pdg.Merge {
		return []pdg.Ref{node}
	}
	var result []pdg.Ref
	for _, edge := range g.Edges {
		if edge.Kind == pdg.Data && edge.Target == node {
			result = append(result, reachingProducers(g, edge.Source)...)
		}
	}
	return result
}

// Display def/use fixtures at their consuming source operation, while read
// occurrence fixtures below check the newly decomposed intermediate entities.
func readConsumer(g pdg.Graph, node pdg.Ref) pdg.Ref {
	if g.Nodes[node-1].Category != pdg.Read {
		return node
	}
	for _, edge := range g.Edges {
		if edge.Source == node && g.Tables.Text(edge.Subkind) == "scalar-value" {
			return edge.Target
		}
	}
	return node
}

func readOperandPosition(g pdg.Graph, definition pdg.Edge) pdg.Position {
	if g.Nodes[definition.Target-1].Category != pdg.Read {
		return definition.Position
	}
	for _, edge := range g.Edges {
		if edge.Source == definition.Target && g.Tables.Text(edge.Subkind) == "scalar-value" {
			return edge.Position
		}
	}
	return definition.Position
}
