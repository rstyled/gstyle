package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestInvalidGraphDoesNotProduceEmptySuccess(t *testing.T) {
	if _, err := Analyze(map[string]*pdg.Graph{"bad": {}}, Models{}); err == nil {
		t.Fatal("invalid graph accepted as an empty analysis")
	}
}
func TestMissingGraphDoesNotInventFlow(t *testing.T) {
	findings, err := Analyze(map[string]*pdg.Graph{"missing": nil}, Models{})
	if err != nil || len(findings) != 0 {
		t.Fatalf("missing graph: %v %v", findings, err)
	}
}
