package patterns

import (
	"fmt"
	"strings"
)

// applyPatternReviews removes only candidates reviewed at every site and keeps
// their complete evidence and score in the separately rendered review list.
func applyPatternReviews(mined *Mined, facts []*FuncFacts) {
	active := make([]Candidate, 0, len(mined.Candidates))
	for _, candidate := range mined.Candidates {
		reason := candidateReview(candidate, facts)
		if reason == "" {
			active = append(active, candidate)
			continue
		}
		reviewed := candidate
		mined.Suppressed = append(mined.Suppressed, Suppressed{
			Reason:    "Reviewed " + string(candidate.Kind) + ": " + reason,
			Sites:     candidate.Sites,
			Candidate: &reviewed,
		})
	}
	mined.Candidates = active
}

func candidateReview(candidate Candidate, facts []*FuncFacts) string {
	if len(candidate.Sites) == 0 {
		return ""
	}
	var reasons []string
	for _, site := range candidate.Sites {
		reason := siteReview(site, candidate.Kind, facts)
		if reason == "" {
			return ""
		}
		reasons = append(reasons, fmt.Sprintf("%s: %s", site.Name, reason))
	}
	return strings.Join(reasons, "; ")
}

func siteReview(site Site, kind CandidateKind, facts []*FuncFacts) string {
	for _, fact := range facts {
		if sameReviewedFunction(fact, site) {
			return fact.ReviewReasons[string(kind)]
		}
	}
	return ""
}

func sameReviewedFunction(fact *FuncFacts, site Site) bool {
	if fact.ID != "" {
		if fact.ID == site.ID || fact.ID == site.Name {
			return true
		}
	}
	return site.Line > 0 && site.Path == fact.Path && site.Line == fact.Line
}
