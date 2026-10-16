package patterns

import (
	"encoding/json"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns/outputcheck"
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

func TestUncheckedOutputCandidateProjection(t *testing.T) {
	facts := MiningFacts{ID: "p.Report", Name: "Report", Path: "a.go", ErrorCheckSites: []ErrorCheckSite{
		{FuncID: "different-owner", Line: 5, Callee: "alias.Fprintln", DiscardedOutput: true},
		{Line: 6, Callee: "alias.Fprintln", Checked: true},
	}}
	payloads := outputcheck.Report(outputcheck.Function{Path: "a.go", ID: "p.Report", Name: "Report"}, facts.ErrorCheckSites, nil)
	candidates := uncheckedOutputCandidates([]*MiningFacts{&facts})
	if len(payloads) != 1 || len(candidates) != 1 {
		t.Fatalf("unexpected findings: payloads=%+v candidates=%+v", payloads, candidates)
	}
	want, got := payloads[0], candidates[0]
	if string(got.Kind) != want.Kind || got.ScoreMilli != want.ScoreMilli || got.Observation != want.Observation || got.Inference != want.Inference {
		t.Fatalf("candidate changes proved payload: got=%+v want=%+v", got, want)
	}
	if len(got.CounterEvidence) != 1 || got.CounterEvidence[0] != want.CounterEvidence || len(got.Sites) != 1 {
		t.Fatalf("candidate drops payload evidence: %+v", got)
	}
	location := got.Sites[0]
	if location != (Site{Path: want.Path, Line: want.Line, EndLine: want.EndLine, ID: want.ID, Name: want.Name}) {
		t.Fatalf("candidate changes proved location: %+v", got)
	}
	if got.FixSpec != nil || got.Analysis != nil || got.Definitions != nil || got.PossibleRefactor != "" || got.Breakdown != (Breakdown{}) {
		t.Fatalf("advisory invents an action or analysis: %+v", got)
	}
}

func TestFailureOnlyOutputExcludesOccurrenceWithoutErasingFacts(t *testing.T) {
	original := MiningFacts{ID: "f", FailureOnlyOutputSites: []int{0}, ErrorCheckSites: []ErrorCheckSite{
		{Line: 3, Callee: "fmt.Fprint", DiscardedOutput: true},
		{Line: 3, Callee: "fmt.Fprintln", DiscardedOutput: true},
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
	if len(got) != 1 || got[0].Observation != "formatted output error is discarded: fmt.Fprintln" {
		t.Fatalf("remaining finding: %+v", got)
	}
	if len(facts.ErrorCheckSites) != 2 || !facts.ErrorCheckSites[0].DiscardedOutput {
		t.Fatalf("raw discard fact erased: %+v", facts)
	}
}
