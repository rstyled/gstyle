package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestScalarExecutionOrderAdmission(t *testing.T) {
	cases := []struct {
		function string
		status   pdg.Status
	}{
		{`func f(x int) int { return x }`, pdg.Supported},
		{`func f(x int) int { x++; return -x }`, pdg.Supported},
		{`func f(x,y bool) bool { return x && !y }`, pdg.Supported},
		{`func f(x int) int { return x+x }`, pdg.Approximated},
		{`func f(x int) int { x+=1; return x }`, pdg.Approximated},
		{`func f(x,y int) int { x,y=y,x; return x }`, pdg.Approximated},
		{`func f(x int) (a,b int) { return }`, pdg.Approximated},
		{`func f(x int) int { return x; x=1+2; return x }`, pdg.Approximated},
	}
	for _, tc := range cases {
		t.Run(tc.function, func(t *testing.T) {
			graph := parseIR(t, "package example; "+tc.function, pdg.NewBuilder())
			for _, capability := range graph.Capabilities {
				if graph.Tables.Text(capability.Name) != "execution_dependencies" {
					continue
				}
				if got := graph.Tables.Evidence[capability.Evidence-1].Status; got != tc.status {
					t.Fatalf("execution status=%v; want %v", got, tc.status)
				}
			}
			for _, edge := range graph.Edges {
				if edge.Kind == pdg.Execution && graph.Tables.Evidence[edge.Evidence-1].Status != tc.status {
					t.Fatal("execution edge and capability disagree")
				}
			}
			if graph.Conformance.Status == pdg.Conformant {
				t.Fatal("ordering alone must not certify extraction")
			}
		})
	}
}
