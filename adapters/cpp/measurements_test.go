package cpp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestCPPTrivialWrappersRemainMeasurements(t *testing.T) {
	report := featureExtraction(t).Patterns()
	for _, candidate := range report.Candidates {
		switch candidate.Kind {
		case patterns.ThinSliceKind, patterns.BarrierSliceKind, patterns.ChopKind, patterns.DependenceCluster:
			t.Fatalf("size alone produces a refactor candidate: %+v", candidate)
		}
	}
	want := map[string]int{"tiny_wrapper": 1, "arithmetic": 0, "with_call": 1}
	for _, measurement := range report.Measurements {
		name, _, _ := strings.Cut(measurement.Site.Name, ":")
		calls, ok := want[name]
		if !ok || measurement.Kind != patterns.ThinSliceKind {
			continue
		}
		if measurement.Calls != calls || measurement.GraphNodes < 10 || len(measurement.SourceLines) != 1 {
			t.Fatalf("C++ lowering or operators become function calls: %+v", measurement)
		}
		total := 0
		for _, count := range measurement.NodeKinds {
			total += count
		}
		if total != measurement.GraphNodes || measurement.NodeKinds["call"] != measurement.Calls {
			t.Fatalf("measurement breakdown does not reconcile: %+v", measurement)
		}
		if measurement.SeedNode == "" || measurement.Analysis == nil || measurement.Analysis.Status != "complete" {
			t.Fatalf("measurement loses its evidence: %+v", measurement)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("fixtures do not exercise graph-size thresholds: %v", want)
	}
	text := patterns.Text(report)
	if !strings.Contains(text, "dependency measurements (context only") || strings.Contains(text, "consider simplifying") {
		t.Fatal("text report treats measurements as refactor advice")
	}
	data, err := patterns.JSON(report)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip patterns.PatternsReport
	if err := json.Unmarshal([]byte(data), &roundTrip); err != nil {
		t.Fatal(err)
	}
	if len(roundTrip.Measurements) != len(report.Measurements) {
		t.Fatal("JSON loses dependency measurements")
	}
}
