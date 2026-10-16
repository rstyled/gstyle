package patterns

import (
	"fmt"
	"testing"
)

func TestMeasurementsDoNotProduceFindings(t *testing.T) {
	facts := []*MiningFacts{sliceTestFact("pkg.first", chainPdg(25))}
	// Establish that the fixture crosses every measurement threshold. A passing
	// empty report must not come from an empty or too-small graph.
	if len(FindBarrierSlices(facts)) == 0 || len(FindThinSlices(facts)) == 0 || len(FindChops(facts)) == 0 {
		t.Fatal("fixture does not exercise the measurement rules")
	}
	report := RunMining(facts, Options{})
	for _, c := range report.Candidates {
		if c.Kind == BarrierSliceKind || c.Kind == ThinSliceKind || c.Kind == ChopKind {
			t.Fatalf("measurement became a finding: %+v", c)
		}
	}
}

func TestStatisticalDeviationDoesNotEstablishACallContract(t *testing.T) {
	var facts []*MiningFacts
	var sites []ErrorCheckSite
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("pkg.f%d", i)
		site := ErrorCheckSite{FuncID: id, Callee: "fetch", Checked: i != 9}
		sites = append(sites, site)
		facts = append(facts, &MiningFacts{ID: id, Name: id, Path: "fixture.go", ErrorCheckSites: []ErrorCheckSite{site}})
	}
	beliefs := MineErrorBeliefs(sites)
	if len(FindBeliefViolations(sites, beliefs)) != 1 {
		t.Fatal("fixture must contain a high-frequency deviation")
	}
	report := RunMining(facts, Options{})
	for _, c := range report.Candidates {
		if c.Kind == DeviantBehavior {
			t.Fatalf("frequency became an error-handling requirement: %+v", c)
		}
	}
}
