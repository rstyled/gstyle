package patterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/taintflow"
)

func TestTaintWorkLimitMarksCoverageUnknown(t *testing.T) {
	facts := []*MiningFacts{{Canonical: &pdg.Graph{}}}
	before := taintQueryCoverage(facts, nil)
	after := taintQueryCoverage(facts, taintflow.ErrWorkLimit)
	for index, row := range after {
		if row.Kind != TaintFlowKind {
			if row != before[index] {
				t.Fatal("taint work limit changes another analysis's coverage")
			}
			continue
		}
		if row.Complete != 0 || row.Partial != 0 || row.Unknown != 1 || row.Reason != taintflow.ErrWorkLimit.Error() {
			t.Fatalf("limited taint coverage appears complete: %+v", row)
		}
	}
}
