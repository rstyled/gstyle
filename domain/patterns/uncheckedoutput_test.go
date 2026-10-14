package patterns

import (
	"encoding/json"
	"testing"
)

func TestUncheckedOutputFactsRoundTripAndSuppression(t *testing.T) {
	original := MiningFacts{ID: "p.Report", Name: "Report", Path: "report.go", ErrorCheckSites: []ErrorCheckSite{
		{Callee: "fmt.Fprintf", Line: 7, DiscardedOutput: true},
		{Callee: "fmt.Fprintf", Line: 8, Checked: true},
	}}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var facts MiningFacts
	if err := json.Unmarshal(encoded, &facts); err != nil {
		t.Fatal(err)
	}
	got := uncheckedOutputCandidates([]*MiningFacts{&facts})
	if len(got) != 1 || got[0].Sites[0].Line != 7 || got[0].Sites[0].Path != "report.go" || got[0].FixSpec != nil {
		t.Fatalf("bad advisory: %+v", got)
	}
	facts.SuppressedKinds = []string{"unchecked_output"}
	if got := uncheckedOutputCandidates([]*MiningFacts{&facts}); len(got) != 0 {
		t.Fatalf("suppression ignored: %+v", got)
	}
}

func TestUncheckedOutputSelectionPreservesOccurrences(t *testing.T) {
	facts := []*MiningFacts{
		{ID: "p.First", Name: "First", Path: "a.go", ErrorCheckSites: []ErrorCheckSite{
			{Line: 3, Callee: "fmt.Fprint", DiscardedOutput: true},
			{Line: 4, Callee: "fmt.Fprint", Checked: true},
			{Line: 3, Callee: "fmt.Fprint", DiscardedOutput: true},
		}},
		{ID: "p.Hidden", SuppressedKinds: []string{"unchecked_output"}, ErrorCheckSites: []ErrorCheckSite{{Line: 8, DiscardedOutput: true}}},
		{ID: "p.Last", Name: "Last", Path: "b.go", ErrorCheckSites: []ErrorCheckSite{{Line: 9, Callee: "fmt.Fprintln", DiscardedOutput: true}}},
	}
	got := uncheckedOutputCandidates(facts)
	if len(got) != 3 {
		t.Fatalf("got %d findings, want 3: %+v", len(got), got)
	}
	for i, want := range []Site{
		{Path: "a.go", Line: 3, EndLine: 3, ID: "p.First", Name: "First"},
		{Path: "a.go", Line: 3, EndLine: 3, ID: "p.First", Name: "First"},
		{Path: "b.go", Line: 9, EndLine: 9, ID: "p.Last", Name: "Last"},
	} {
		if len(got[i].Sites) != 1 || got[i].Sites[0] != want || got[i].Kind != UncheckedOutput || got[i].ScoreMilli != 800 || got[i].FixSpec != nil {
			t.Fatalf("finding %d loses metadata or advisory policy: %+v", i, got[i])
		}
	}
}
