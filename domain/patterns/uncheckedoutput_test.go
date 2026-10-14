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
