package quality

import "testing"

func TestRecursiveContextRequiresEveryReachableBody(t *testing.T) {
	parameter := 0
	for _, provenance := range []Provenance{Local, External, Unknown} {
		for _, leaf := range []Call{{}, {Callee: "missing", Local: true, Line: 6}, {Callee: "callback", Dynamic: true, Line: 6}, {Callee: "fmt.Println", Line: 6}} {
			f := fact("caller", 1)
			f.Calls = []Call{{Callee: "loop", Local: true, Line: 2, StorageArgs: []Storage{{Provenance: provenance}}}}
			loop := Helper{Name: "loop", Params: 1,
				Mutations: []Mutation{{Root: "visited", Line: 3, Provenance: External, Parameter: &parameter}},
				Calls:     []Call{{Callee: "next", Local: true, Line: 4, StorageArgs: []Storage{{Provenance: External, Parameter: &parameter}}}}}
			next := Helper{Name: "next", Params: 1, Calls: []Call{{Callee: "loop", Local: true, Line: 5, StorageArgs: []Storage{{Provenance: External, Parameter: &parameter}}}}}
			if leaf.Callee != "" {
				next.Calls = append(next.Calls, leaf)
			}
			f.Helpers = []Helper{loop, next}
			got := evaluate(t, f, DefaultConfig()).Functions[0]
			want := provenance == Local && leaf.Callee == ""
			if (got.Complete && len(got.Effects) == 0) != want {
				t.Fatalf("storage=%s leaf=%+v: %+v", provenance, leaf, got)
			}
		}
	}
}

func TestRecursiveContextChecksDistinctStorageBindings(t *testing.T) {
	parameter := 0
	initial := Call{Callee: "loop", Local: true, StorageArgs: []Storage{{Provenance: Local}}}
	r := callResolver{helpers: map[string]Helper{"loop": {
		Name: "loop", Params: 1,
		Mutations: []Mutation{{Provenance: External, Parameter: &parameter}},
		Calls:     []Call{{Callee: "loop", Local: true, StorageArgs: []Storage{{Provenance: External}}}},
	}}}
	if r.contextReadOnly(initial) {
		t.Fatal("recursive call changes owned storage to shared storage")
	}
}

func TestRecursiveContextKeepsBudgetAndPolicyBoundaries(t *testing.T) {
	parameter := 0
	initial := Call{Callee: "loop", Local: true, StorageArgs: []Storage{{Provenance: Local}}}
	helper := Helper{Name: "loop", Params: 1,
		Mutations: []Mutation{{Provenance: External, Parameter: &parameter}},
		Calls:     []Call{{Callee: "loop", Local: true, StorageArgs: []Storage{{Provenance: External, Parameter: &parameter}}}, {Callee: "project.leaf"}},
	}
	r := callResolver{helpers: map[string]Helper{"loop": helper}, prefixes: []Prefix{{Path: "project.leaf", Kind: None}}}
	if !r.contextReadOnly(initial) {
		t.Fatal("owned cycle with explicit read-only leaf is unproved")
	}
	r.prefixes = append(r.prefixes, Prefix{Path: "project.leaf", Kind: IO})
	if r.contextReadOnly(initial) {
		t.Fatal("policy override hidden by recursive proof")
	}
	seen := make(map[string]bool, localContextLimit)
	for i := range localContextLimit {
		seen[string(rune(i))] = true
	}
	if _, known := r.readOnlyContext(initial, seen); known {
		t.Fatal("context budget exceeded")
	}
}
