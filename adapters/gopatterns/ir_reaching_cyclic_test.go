package gopatterns

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCyclicScalarReachingDefinitions(t *testing.T) {
	cases := []struct {
		name, function string
		want           []string
	}{
		{"zero-or-many", `func f(x int, c bool) int { for c { x = 1 }; return x }`, []string{"c bool -> for c { x = 1 } : c", "x int -> return x : x", "x = 1 -> return x : x"}},
		{"loop-read", `func f(x int) int { for x < 2 { x++ }; return x }`, []string{"x int -> x < 2 : x", "x++ -> x < 2 : x", "x int -> x++ : x", "x++ -> x++ : x", "x int -> return x : x", "x++ -> return x : x"}},
		{"killed-before-read", `func f(x int, c bool) int { for c { x = 1; x = x + 2 }; return x }`, []string{"c bool -> for c { x = 1; x = x + 2 } : c", "x = 1 -> x + 2 : x", "x int -> return x : x", "x = x + 2 -> return x : x"}},
		{"break", `func f(x int, c bool) int { for { if c { break }; x = 1 }; return x }`, []string{"c bool -> if c { break } : c", "x int -> return x : x", "x = 1 -> return x : x"}},
		{"continue-kills", `func f(x int, c bool) int { for c { x = 1; continue; x = 2 }; return x }`, []string{"c bool -> for c { x = 1; continue; x = 2 } : c", "x int -> return x : x", "x = 1 -> return x : x"}},
		{"backward-goto", `func f(x int, c bool) int { again: x++; if c { goto again }; return x }`, []string{"x int -> x++ : x", "x++ -> x++ : x", "c bool -> if c { goto again } : c", "x++ -> return x : x"}},
		{"forward-goto", `func f(x int) int { goto done; x = 1; done: return x }`, []string{"x int -> return x : x"}},
		{"named-bare-return", `func f(c bool) (x int) { for c { x++ }; return }`, []string{"c bool -> for c { x++ } : c", "x int -> x++ : x", "x++ -> x++ : x", "x int -> return : x", "x++ -> return : x"}},
		{"for-post", `func f(x int) int { for i:=0; i<x; i++ { x=i }; return x }`, []string{"i:=0 -> i<x : i", "i++ -> i<x : i", "x int -> i<x : x", "x=i -> i<x : x", "i:=0 -> x=i : i", "i++ -> x=i : i", "i:=0 -> i++ : i", "i++ -> i++ : i", "x int -> return x : x", "x=i -> return x : x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package example\n" + tc.function
			g := parseIR(t, source, pdg.NewBuilder())
			if err := pdg.Validate(&g); err != nil {
				t.Fatal(err)
			}
			if reachingCapability(g) != pdg.Supported {
				t.Fatal("missing exact reaching definitions")
			}
			got := reachingDescriptions(t, g, source)
			slices.Sort(tc.want)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v; want %v", got, tc.want)
			}
			if g.NormalFlow != nil {
				t.Fatal("cyclic source admitted under the acyclic matching profile")
			}
			for _, edge := range g.Edges {
				if g.Tables.Text(edge.Subkind) == "mining-value" {
					t.Fatal("first-binding edge remains")
				}
			}
			again := parseIR(t, source, pdg.NewBuilder())
			if !reflect.DeepEqual(g, again) {
				t.Fatal("unstable extraction")
			}
		})
	}
}

func TestExtendedScalarReturnPaths(t *testing.T) {
	cases := []struct {
		name, body string
		want       []string
	}{
		{"false-loop", `for false { x=1 }; return x`, []string{"x int"}},
		{"constant-break", `for true { x=1; break }; return x`, []string{"x=1"}},
		{"short-circuit", `for false && c { x=1 }; return x`, []string{"x int"}},
		{"switch-join", `switch x { case 1: x=2; case 2: x=3 }; return x`, []string{"x int", "x=2", "x=3"}},
		{"switch-fallthrough", `switch c { case true: x=2; fallthrough; default: x=3 }; return x`, []string{"x=3"}},
		{"nested-loop", `for c { for c { x=1; break }; x=2 }; return x`, []string{"x int", "x=2"}},
		{"shadow", `for c { x:=1; x++ }; return x`, []string{"x int"}},
		{"named-result", `for c { x=1 }; return x`, []string{"x int", "x=1"}},
		{"infinite", `for {}; return x`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package example; func f(x int,c bool) int { " + tc.body + " }"
			graph := parseIR(t, source, pdg.NewBuilder())
			var got []string
			for _, edge := range graph.Edges {
				if graph.Tables.Text(edge.Subkind) != "reaching-definition" || graph.Nodes[edge.Target-1].Category == pdg.Merge {
					continue
				}
				target := readConsumer(graph, edge.Target)
				if graph.Nodes[target-1].Category != pdg.Return {
					continue
				}
				for _, producer := range reachingProducers(graph, edge.Source) {
					span := graph.Spans[graph.Nodes[producer-1].Span-1]
					got = append(got, source[span.Start:span.End])
				}
			}
			slices.Sort(got)
			slices.Sort(tc.want)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("return producers got %v want %v", got, tc.want)
			}
			if tc.name == "infinite" {
				for _, edge := range graph.Edges {
					if graph.Tables.Text(edge.Subkind) == "return-value" {
						t.Fatal("unreachable return transfers a value")
					}
				}
			}
		})
	}
}

func TestExtendedScalarRejectsUnsupportedEffectsInWholeBody(t *testing.T) {
	for _, source := range []string{
		`func f(x int) int { for x<2 { println(x); x++ }; return x }`,
		`func f(x int) int { for x<2 { defer func(){x++}(); x++ }; return x }`,
		`func f(x int) int { for x<2 { p:=&x; *p=2 }; return x }`,
		`func f(x int) int { for x<2 { go func(){x++}(); x++ }; return x }`,
		`func f(x int) int { for x<2 { return x }; println(x); return x }`,
		`func f(xs []int) int { x:=0; for _,v:=range xs { x+=v }; return x }`,
	} {
		graph := parseIR(t, "package example; "+source, pdg.NewBuilder())
		if reachingCapability(graph) == pdg.Supported {
			t.Fatalf("unsupported body claims exact definitions: %s", source)
		}
		for _, edge := range graph.Edges {
			if edge.Definition != 0 && graph.Tables.Evidence[edge.Evidence-1].Status == pdg.Supported {
				t.Fatalf("partial body emits exact definitions: %s", source)
			}
		}
	}
}
