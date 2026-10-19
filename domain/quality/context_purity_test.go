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
			complete := provenance != Unknown && (leaf.Callee == "" || leaf.Callee == "fmt.Println")
			if got.Complete != complete {
				t.Fatalf("storage=%s leaf=%+v: complete=%v want=%v", provenance, leaf, got.Complete, complete)
			}
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
	if len(r.recursiveCall(initial).effects) == 0 {
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
	if len(r.recursiveCall(initial).effects) != 0 {
		t.Fatal("owned cycle with explicit read-only leaf is unproved")
	}
	r.prefixes = append(r.prefixes, Prefix{Path: "project.leaf", Kind: IO})
	if len(r.recursiveCall(initial).effects) == 0 {
		t.Fatal("policy override hidden by recursive proof")
	}
	seen := make(map[string]bool, localContextLimit)
	for i := range localContextLimit {
		seen[string(rune(i))] = true
	}
	walk := recursiveSummary{resolver: &r, seen: seen}
	if got := walk.context(initial); len(got.effects) == 0 || got.effects[0].Kind != UnknownEffect {
		t.Fatal("context budget exceeded")
	}
}

func TestRecursiveSummaryBoundsPreserveKnownEffects(t *testing.T) {
	effects := make([]Effect, localEffectLimit+1)
	for i := range effects {
		effects[i] = Effect{Kind: IO, Detail: "write", Line: 2}
	}
	calls := make([]Call, localContextLimit+1)
	for i := range calls {
		calls[i] = Call{Callee: "loop", Local: true, Line: 3}
	}
	for _, helper := range []Helper{
		{Name: "loop", Effects: effects, Calls: []Call{{Callee: "loop", Local: true, Line: 3}}},
		{Name: "loop", Effects: effects[:1], Calls: calls},
	} {
		r := callResolver{helpers: map[string]Helper{"loop": helper}}
		got := r.recursiveCall(Call{Callee: "loop", Local: true, Site: 17, Line: 1})
		if got.unclassified != 1 || got.effects[0].Kind != IO || got.effects[len(got.effects)-1].Kind != UnknownEffect || len(got.effects) > localEffectLimit {
			t.Fatalf("recursive summary budget loses evidence: %+v", got)
		}
	}
}
