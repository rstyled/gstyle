package quality

import "testing"

func TestContainerOwnershipBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		call    Call
		kind    Kind
		invalid bool
	}{
		{name: "owned", call: Call{Callee: "slices.Sort[int]", MutationTarget: Local}, kind: None},
		{name: "external", call: Call{Callee: "slices.Sort", MutationTarget: External}, kind: MutationEffect},
		{name: "unknown", call: Call{Callee: "slices.Sort", MutationTarget: Unknown}, kind: MutationEffect},
		{name: "legacy", call: Call{Callee: "slices.Sort"}, kind: MutationEffect},
		{name: "unmodeled", call: Call{Callee: "custom.Sort", MutationTarget: Local}, invalid: true},
		{name: "expanded inline callback", call: Call{Callee: "slices.SortFunc", MutationTarget: Local, CallbackExpanded: true}, kind: None},
		{name: "legacy callback", call: Call{Callee: "slices.SortFunc"}, kind: MutationEffect},
		{name: "wrong expanded callee", call: Call{Callee: "maps.Copy", CallbackExpanded: true}, invalid: true},
		{name: "interface callback", call: Call{Callee: "sort.Sort", CallbackExpanded: true}, invalid: true},
		{name: "dynamic expanded callback", call: Call{Callee: "slices.SortFunc", CallbackExpanded: true, Dynamic: true}, invalid: true},
		{name: "callback", call: Call{Callee: "slices.SortFunc", MutationTarget: Local}, invalid: true},
		{name: "dynamic", call: Call{Callee: "slices.Sort", MutationTarget: Local, Dynamic: true}, invalid: true},
		{name: "bad provenance", call: Call{Callee: "maps.Copy", MutationTarget: "invalid"}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := fact("f", 1)
			tc.call.Line = 2
			f.Calls = []Call{tc.call}
			r, err := Evaluate([]Function{f}, DefaultConfig())
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid ownership accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			effects := r.Functions[0].Effects
			if tc.kind == None {
				if len(effects) != 0 {
					t.Fatal(effects)
				}
				return
			}
			if len(effects) != 1 || effects[0].Kind != tc.kind {
				t.Fatalf("effects: %+v", effects)
			}
		})
	}
}
