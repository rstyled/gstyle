package patterns

import (
	"encoding/json"
	"testing"
)

func TestDiscardedOutputDoesNotEstablishRequiredOutput(t *testing.T) {
	original := MiningFacts{ID: "p.Report", Name: "Report", Path: "report.go", SuccessReturnOutputSites: []int{0}, ErrorCheckSites: []ErrorCheckSite{
		{Callee: "fmt.Fprintln", Line: 7, DiscardedOutput: true},
	}}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var facts MiningFacts
	if err = json.Unmarshal(encoded, &facts); err != nil {
		t.Fatal(err)
	}
	report := RunMining([]*MiningFacts{&facts}, Options{})
	for _, candidate := range report.Candidates {
		if candidate.Kind == UncheckedOutput {
			t.Fatalf("inferred required output without contract evidence: %+v", candidate)
		}
	}
	if len(facts.ErrorCheckSites) != 1 || !facts.ErrorCheckSites[0].DiscardedOutput || len(facts.SuccessReturnOutputSites) != 1 {
		t.Fatalf("raw structural evidence lost: %+v", facts)
	}
}
