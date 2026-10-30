package patterns

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCloneCandidatesKeepFunctionIdentityAndRange(t *testing.T) {
	pool := pdg.NewBuilder()
	facts := []*MiningFacts{
		{ID: "reader", Name: "decode", Path: "reader.go", Line: 929, EndLine: 954,
			Canonical: &pdg.Graph{Tables: pool.Tables, Function: pdg.Function{QualifiedName: pool.Known("Reader.decode")}}},
		{ID: "our-reader", Name: "decode", Path: "reader.go", Line: 1900, EndLine: 1925,
			Canonical: &pdg.Graph{Tables: pool.Tables, Function: pdg.Function{QualifiedName: pool.Known("OurReader.decode")}}},
	}
	want := []Site{
		{ID: "reader", Name: "Reader.decode", Path: "reader.go", Line: 929, EndLine: 954},
		{ID: "our-reader", Name: "OurReader.decode", Path: "reader.go", Line: 1900, EndLine: 1925},
	}
	for name, build := range map[string]func([][]string, []*MiningFacts) []Candidate{
		"clone": CCGraphCloneCandidates, "divergent clone": InconsistentCloneCandidates,
	} {
		t.Run(name, func(t *testing.T) {
			candidates := build([][]string{{"reader", "missing", "our-reader"}}, facts)
			if len(candidates) != 1 || !reflect.DeepEqual(candidates[0].Sites, want) {
				t.Fatalf("clone loses function identity or extent: %+v", candidates)
			}
		})
	}
	observation := CCGraphCloneCandidates([][]string{{"reader", "our-reader"}}, facts)[0].Observation
	if !strings.Contains(observation, "2 function bodies") || strings.Contains(observation, "decode, decode") {
		t.Fatalf("clone observation repeats ambiguous method names: %s", observation)
	}
	if facts[0].Name != "decode" || facts[1].Name != "decode" {
		t.Fatal("display names change the matching inputs")
	}
}

func TestCloneSiteKeepsNameWhenQualificationIsUnavailable(t *testing.T) {
	pool := pdg.NewBuilder()
	for _, graph := range []*pdg.Graph{
		nil,
		{},
		{Tables: pool.Tables, Function: pdg.Function{QualifiedName: pool.Missing(pdg.Unknown, "not recorded")}},
		{Tables: pool.Tables, Function: pdg.Function{QualifiedName: pool.Known("")}},
	} {
		fact := &MiningFacts{ID: "f", Name: "decode", Path: "reader.go", Line: 20, EndLine: 30, Canonical: graph}
		if got := cloneSite(fact); got != makeSite(fact) {
			t.Fatalf("unavailable qualification replaces the recorded site: %+v", got)
		}
	}
}
