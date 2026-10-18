package quality

import (
	"strings"
	"testing"
)

func TestStorageFactValidation(t *testing.T) {
	index := 0
	receiver := -1
	negative := -2
	for _, tc := range []struct {
		name   string
		change func(*Function)
	}{
		{"missing formal", func(f *Function) {
			f.Mutations = []Mutation{{Root: "p", Line: 2, Provenance: External, Parameter: &index}}
		}},
		{"missing receiver", func(f *Function) {
			f.Mutations = []Mutation{{Root: "p", Line: 2, Provenance: External, Parameter: &receiver}}
		}},
		{"local formal contradiction", func(f *Function) {
			f.Params = 1
			f.Mutations = []Mutation{{Root: "p", Line: 2, Provenance: Local, Parameter: &index}}
		}},
		{"bad storage provenance", func(f *Function) {
			f.Calls = []Call{{Callee: "helper", Line: 2, StorageArgs: []Storage{{Provenance: "invalid"}}}}
		}},
		{"bad storage formal", func(f *Function) {
			f.Calls = []Call{{Callee: "helper", Line: 2, StorageArgs: []Storage{{Provenance: External, Parameter: &negative}}}}
		}},
		{"local source contradiction", func(f *Function) {
			f.Params = 1
			f.Calls = []Call{{Callee: "helper", Line: 2, StorageArgs: []Storage{{Provenance: Local, Parameter: &index}}}}
		}},
		{"missing helper formal", func(f *Function) {
			f.Helpers = []Helper{{Name: "helper", Mutations: []Mutation{{Root: "p", Line: 2, Provenance: External, Parameter: &index}}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := fact("caller", 1)
			tc.change(&f)
			if _, err := Evaluate([]Function{f}, DefaultConfig()); err == nil {
				t.Fatal("invalid binding accepted")
			}
		})
	}
}

func TestHelperOwnershipContextLimitRemainsUnknown(t *testing.T) {
	formal := 0
	f := fact("caller", 1)
	f.Helpers = []Helper{{Name: "helper", Params: 13, Mutations: []Mutation{{Root: "p", Line: 2, Provenance: External, Parameter: &formal}}}}
	for context := 0; context < 8192; context++ {
		storage := make([]Storage, 13)
		for parameter := range storage {
			provenance := Local
			if context&(1<<parameter) != 0 {
				provenance = External
			}
			storage[parameter] = Storage{Provenance: provenance}
		}
		f.Calls = append(f.Calls, Call{Callee: "helper", Line: 2, Local: true, StorageArgs: storage})
	}
	report, err := Evaluate([]Function{f}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	result := report.Functions[0]
	if result.Complete {
		t.Fatal("exhausted ownership contexts reported complete")
	}
	found := false
	for _, effect := range result.Effects {
		found = found || strings.Contains(effect.Detail, "local context limit")
	}
	if !found {
		t.Fatal("context exhaustion lost its unknown evidence")
	}
	if len(result.Effects) > localEffectLimit {
		t.Fatalf("effect bound exceeded: %d", len(result.Effects))
	}
}
