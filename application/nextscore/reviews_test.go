package nextscore

import (
	"github.com/shanejonas/cyclo/domain/patterns"
	"testing"
)

func TestReviewedCandidatesRetainScoreWeight(t *testing.T) {
	candidate := patterns.Candidate{Kind: patterns.ChopKind, ScoreMilli: 800}
	before := computeScore(&patterns.PatternsReport{Candidates: []patterns.Candidate{candidate}})
	after := computeScore(&patterns.PatternsReport{Suppressed: []patterns.Suppressed{{Candidate: &candidate, Reason: "reviewed"}}})
	if before != after || after != 92 {
		t.Fatalf("review changed score: %d -> %d", before, after)
	}
}
