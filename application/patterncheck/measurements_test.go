package patterncheck

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestDependencyMeasurementsAreNotSARIFFindings(t *testing.T) {
	report := &patterns.PatternsReport{
		Candidates:   []patterns.Candidate{{Kind: patterns.GuardClause, Sites: []patterns.Site{{Path: "guard.cpp", Line: 8}}}},
		Measurements: []patterns.DependencyMeasurement{{Kind: patterns.ThinSliceKind, GraphNodes: 11, Calls: 1, Site: patterns.Site{Path: "guard.cpp", Line: 1}}},
	}
	var output bytes.Buffer
	if err := writeSARIF(&output, report); err != nil {
		t.Fatal(err)
	}
	var sarif struct {
		Runs []struct {
			Results []struct {
				RuleID string `json:"ruleId"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(output.Bytes(), &sarif); err != nil {
		t.Fatal(err)
	}
	if len(sarif.Runs) != 1 || len(sarif.Runs[0].Results) != 1 || sarif.Runs[0].Results[0].RuleID != "guard_clause" {
		t.Fatalf("measurement is exported as a warning: %s", output.String())
	}
}
