package quality

import (
	"fmt"
	"testing"
)

func TestReadOnlyHelpersAgainstReachabilityOracle(t *testing.T) {
	const size = 3
	for edges := 0; edges < 1<<(size*size); edges++ {
		for dirty := 0; dirty < 1<<size; dirty++ {
			helpers := map[string]Helper{}
			reach := [size][size]bool{}
			for from := range size {
				name := fmt.Sprint("h", from)
				helper := Helper{Name: name}
				reach[from][from] = true
				if dirty&(1<<from) != 0 {
					helper.Effects = []Effect{{Kind: []Kind{IO, Global, MutationEffect}[from], Detail: "write", Line: 1}}
				}
				for to := range size {
					if edges&(1<<(from*size+to)) != 0 {
						helper.Calls = append(helper.Calls, Call{Callee: fmt.Sprint("h", to), Local: true})
						reach[from][to] = true
					}
				}
				helpers[name] = helper
			}
			// Independent transitive closure: a helper is read-only iff no reachable
			// node has a direct effect. This covers every graph, including all cycles.
			for via := range size {
				for from := range size {
					for to := range size {
						reach[from][to] = reach[from][to] || reach[from][via] && reach[via][to]
					}
				}
			}
			got := readOnlyHelpers(helpers, nil)
			for root := range size {
				want := true
				for node := range size {
					if reach[root][node] && dirty&(1<<node) != 0 {
						want = false
					}
				}
				resolver := callResolver{helpers: helpers, pure: got, cache: map[string]callSummary{}, active: map[string]bool{}}
				summary := resolver.recursiveCall(Call{Callee: fmt.Sprint("h", root), Local: true})
				if (len(summary.effects) == 0) != want || summary.unclassified != 0 {
					t.Fatalf("edges=%d dirty=%d root=%d: recursive effects=%+v", edges, dirty, root, summary)
				}
				resolved := resolver.call(Call{Callee: fmt.Sprint("h", root), Local: true})
				kinds := map[Kind]bool{}
				for _, effect := range resolved.effects {
					kinds[effect.Kind] = true
				}
				for node, kind := range []Kind{IO, Global, MutationEffect} {
					if kinds[kind] != (reach[root][node] && dirty&(1<<node) != 0) {
						t.Fatalf("edges=%d dirty=%d root=%d: resolved kinds=%v", edges, dirty, root, kinds)
					}
				}
				if kinds[UnknownEffect] || resolved.unclassified != 0 {
					t.Fatalf("known graph becomes unknown: %+v", resolved)
				}
				if got[fmt.Sprint("h", root)] != want {
					t.Fatalf("edges=%d dirty=%d root=%d: read-only=%v want=%v", edges, dirty, root, got, want)
				}
			}
		}
	}
}

func TestReadOnlyRecursionKeepsUnprovedOperations(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mutations []Mutation
		calls     []Call
		effects   []Effect
	}{
		{name: "local", mutations: []Mutation{{Root: "local", Provenance: Local, Line: 1}}},
		{name: "external", mutations: []Mutation{{Root: "parameter", Provenance: External, Line: 1}}},
		{name: "unknown storage", mutations: []Mutation{{Root: "alias", Provenance: Unknown, Line: 1}}},
		{name: "missing body", calls: []Call{{Callee: "missing", Local: true, Line: 1}}},
		{name: "dynamic call", calls: []Call{{Callee: "callback", Dynamic: true, Line: 1}}},
		{name: "panic", effects: []Effect{{Kind: Panic, Detail: "panic", Line: 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := Helper{Name: "loop", Mutations: tc.mutations, Effects: tc.effects, Calls: append([]Call{{Callee: "loop", Local: true, Line: 1}}, tc.calls...)}
			got := readOnlyHelpers(map[string]Helper{"loop": helper}, nil)["loop"]
			if got != (tc.name == "local") {
				t.Fatalf("read-only=%v", got)
			}
		})
	}
}

func TestRecursiveReadOnlyProofRetainsPolicyOverrides(t *testing.T) {
	f := fact("caller", 1)
	f.Calls = []Call{{Callee: "loop", Local: true, Line: 2}}
	f.Helpers = []Helper{{Name: "loop", Calls: []Call{{Callee: "loop", Local: true, Line: 3}, {Callee: "project.leaf", Line: 4}}}}
	config := DefaultConfig()
	config.Prefixes = append(config.Prefixes, Prefix{Path: "project.leaf", Kind: None})
	if got := evaluate(t, f, config).Functions[0]; !got.Complete || len(got.Effects) != 0 {
		t.Fatalf("read-only recursion=%+v", got)
	}
	config.Prefixes = append(config.Prefixes, Prefix{Path: "project.leaf", Kind: IO})
	got := evaluate(t, f, config).Functions[0]
	kinds := map[Kind]bool{}
	for _, effect := range got.Effects {
		kinds[effect.Kind] = true
	}
	if !kinds[IO] || kinds[UnknownEffect] || !got.Complete {
		t.Fatalf("override hidden by recursion proof: %+v", got)
	}
}

func TestReadOnlyProofDoesNotNeedCallExpansion(t *testing.T) {
	f := fact("caller", 1)
	f.Calls = []Call{{Callee: "h0", Local: true, Line: 2}}
	for index := 0; index < localDepthLimit*2; index++ {
		helper := Helper{Name: fmt.Sprint("h", index)}
		if index+1 < localDepthLimit*2 {
			helper.Calls = []Call{{Callee: fmt.Sprint("h", index+1), Local: true, Line: 3}}
		}
		f.Helpers = append(f.Helpers, helper)
	}
	got := evaluate(t, f, DefaultConfig()).Functions[0]
	if !got.Complete || len(got.Effects) != 0 {
		t.Fatalf("finite read-only graph needs expansion: %+v", got)
	}
}
