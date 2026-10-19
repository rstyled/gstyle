package quality

import "testing"

func TestPlainJSONEvidenceOnlyRefinesStaticMarshal(t *testing.T) {
	for _, call := range []Call{
		{Callee: "encoding/json.Unmarshal", Line: 2},
		{Callee: "other", Line: 2},
		{Callee: "encoding/json.Marshal", Line: 2, Dynamic: true},
	} {
		call.StorageArgs = []Storage{{PlainJSON: true, Provenance: Local}}
		f := fact("caller", 1)
		f.Calls = []Call{call}
		got := evaluate(t, f, DefaultConfig()).Functions[0]
		if got.Complete || got.UnclassifiedCalls != 1 {
			t.Fatal("typed argument hides unrelated effect", got)
		}
	}
}

func TestJSONWithoutTypedProofStaysUnknown(t *testing.T) {
	f := fact("caller", 1)
	f.Calls = []Call{{Callee: "encoding/json.Marshal", Line: 2}}
	got := evaluate(t, f, DefaultConfig()).Functions[0]
	if got.Complete || got.UnclassifiedCalls != 1 {
		t.Fatal("old facts claim plain JSON", got)
	}
	f.Calls[0].StorageArgs = []Storage{{PlainJSON: true, Provenance: Local}}
	config := DefaultConfig()
	config.Prefixes = []Prefix{{Path: "encoding/json.Marshal", Kind: UnknownEffect}}
	got = evaluate(t, f, config).Functions[0]
	if got.Complete || got.UnclassifiedCalls != 1 {
		t.Fatal("custom policy replaced", got)
	}
}
