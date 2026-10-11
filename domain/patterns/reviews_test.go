package patterns

import (
	"reflect"
	"strings"
	"testing"
)

func TestReviewedCandidateKeepsEvidenceAndRequiresEverySite(t *testing.T) {
	candidate := Candidate{Kind: Parameterize, ScoreMilli: 700, Observation: "same graph", Sites: []Site{{ID: "a", Name: "a"}, {ID: "b", Name: "b"}}}
	facts := []*FuncFacts{
		{ID: "a", ReviewReasons: map[string]string{"parameterize": "distinct input contract"}},
		{ID: "b", ReviewReasons: map[string]string{}},
	}
	partial := Mined{Candidates: []Candidate{candidate}}
	applyPatternReviews(&partial, facts)
	if len(partial.Candidates) != 1 || len(partial.Suppressed) != 0 {
		t.Fatalf("partial review hid candidate: %+v", partial)
	}
	facts[1].ReviewReasons["parameterize"] = "distinct output contract"
	complete := Mined{Candidates: []Candidate{candidate}}
	applyPatternReviews(&complete, facts)
	if len(complete.Candidates) != 0 || len(complete.Suppressed) != 1 {
		t.Fatalf("complete review: %+v", complete)
	}
	reviewed := complete.Suppressed[0]
	if !reflect.DeepEqual(*reviewed.Candidate, candidate) || !strings.Contains(reviewed.Reason, "distinct output contract") {
		t.Fatalf("lost evidence: %+v", reviewed)
	}
}

func TestPatternReviewIsScopedToKindAndLocation(t *testing.T) {
	facts := []*FuncFacts{{ID: "a", Path: "a.go", Line: 5, ReviewReasons: map[string]string{"chop": "cohesive calculation"}}}
	for _, candidate := range []Candidate{
		{Kind: ThinSliceKind, Sites: []Site{{ID: "a"}}},
		{Kind: ChopKind, Sites: []Site{{Path: "a.go", Line: 6}}},
		{Kind: ChopKind},
	} {
		mined := Mined{Candidates: []Candidate{candidate}}
		applyPatternReviews(&mined, facts)
		if len(mined.Candidates) != 1 {
			t.Fatalf("unrelated review hid candidate: %+v", candidate)
		}
	}
	located := Mined{Candidates: []Candidate{{Kind: ChopKind, Sites: []Site{{Path: "a.go", Line: 5}}}}}
	applyPatternReviews(&located, facts)
	if len(located.Suppressed) != 1 {
		t.Fatalf("source location did not match: %+v", located)
	}
}
