package patterns

import (
	"reflect"
	"testing"
)

func TestDependencyMeasurementsRetainSharedSliceResults(t *testing.T) {
	fact := sliceTestFact("chain", chainPdg(25))
	for _, kind := range []CandidateKind{ThinSliceKind, BarrierSliceKind, ChopKind} {
		measurements := dependencyMeasurements(fact, kind)
		if len(measurements) != 1 || measurements[0].GraphNodes != 25 || measurements[0].Calls != 0 || len(measurements[0].SourceLines) != 25 {
			t.Fatalf("%s loses the dependency context: %+v", kind, measurements)
		}
	}
}

func TestDependencyMeasurementCountsCanonicalCalls(t *testing.T) {
	graph := semanticFlowFixture(t)
	fact := semanticFact(&MiningFacts{ID: "flow", Canonical: graph})
	measurement := measureDependencies(fact, ThinSliceKind, []int{3, 1, 2, 0}, 3)
	if measurement.GraphNodes != 4 || measurement.Calls != 2 || measurement.NodeKinds["assignment"] != 1 || measurement.NodeKinds["read"] != 1 {
		t.Fatalf("unlabelled operations become calls or disappear: %+v", measurement)
	}
	if !reflect.DeepEqual(measurement.SourceLines, []int{1, 2, 3, 4}) || measurement.SeedNode != "sink" {
		t.Fatalf("source evidence is unstable: %+v", measurement)
	}
}

func TestCPPClonesRequireDivergenceBeforeInconsistency(t *testing.T) {
	facts := []*MiningFacts{{ID: "first", Name: "processData", Pdg: ccTestPdg(20, 1)}, {ID: "second", Name: "processDatum", Pdg: ccTestPdg(20, 100)}}
	kinds := map[CandidateKind]int{}
	for _, candidate := range cppCloneCandidates(facts) {
		kinds[candidate.Kind]++
	}
	if kinds[CCGraphClone] != 1 || kinds[InconsistentClone] != 0 {
		t.Fatalf("identical clones become inconsistent: %v", kinds)
	}
	facts = append(facts, &MiningFacts{ID: "third", Name: "processDatas", Pdg: ccTestPdg(22, 200)})
	kinds = map[CandidateKind]int{}
	for _, candidate := range cppCloneCandidates(facts) {
		kinds[candidate.Kind]++
	}
	if kinds[CCGraphClone] != 1 || kinds[InconsistentClone] != 1 {
		t.Fatalf("actual structural divergence is lost: %v", kinds)
	}
}
