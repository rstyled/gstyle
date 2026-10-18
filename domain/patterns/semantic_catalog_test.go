package patterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestMalformedCallableCatalogReportsQueryFailure(t *testing.T) {
	report := Run(nil, Options{Callables: []*pdg.Graph{{}}})
	for _, row := range report.AnalysisCoverage {
		if row.Kind == TaintFlowKind {
			if row.Reason == "" || row.Complete != 0 {
				t.Fatalf("malformed catalog looks successful: %+v", row)
			}
			return
		}
	}
	t.Fatal("missing taint query failure coverage")
}
