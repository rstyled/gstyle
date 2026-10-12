package nextscore

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestIncompleteCoverageAndAdvisoryFixInstructions(t *testing.T) {
	var output bytes.Buffer
	writeAnalysisCoverage(&output, []patterns.AnalysisCoverage{{Kind: patterns.TaintFlowKind, Partial: 1}})
	writeFixInstructions(&output, patterns.Candidate{Kind: patterns.TaintFlowKind})
	text := output.String()
	if !strings.Contains(text, "coverage is incomplete") || !strings.Contains(text, "no automatic fix") {
		t.Fatalf("missing analysis limits: %s", text)
	}
	if strings.Contains(text, "Fix with:") {
		t.Fatalf("advisory finding advertised an auto-fix: %s", text)
	}
}

func TestFixInstructionsUseVerifiedSpecFile(t *testing.T) {
	var output bytes.Buffer
	candidate := patterns.Candidate{Kind: patterns.GuardClause, FixSpec: &patterns.FixSpec{Kind: patterns.GuardClause, File: "selected.go"}}
	writeFixInstructions(&output, candidate)
	if !strings.Contains(output.String(), "guard_clause selected.go") {
		t.Fatal(output.String())
	}
}
