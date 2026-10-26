package patterns

import "strconv"

type ConstantBoolReturnHit struct {
	Line    int
	EndLine int
	Value   bool
}

func constantBoolReturnCandidates(facts []*MiningFacts) []Candidate {
	var out []Candidate
	for _, function := range facts {
		if isSuppressed(function.SuppressedKinds, ConstantBoolReturn) {
			continue
		}
		for _, hit := range function.ConstantBoolReturns {
			out = append(out, Candidate{
				Kind: ConstantBoolReturn, ScoreMilli: 800,
				Observation:      "both explicit return paths at this conditional return " + strconv.FormatBool(hit.Value),
				Inference:        "the condition does not change the returned boolean; review the intended predicate",
				PossibleRefactor: "check the branch results against the intended behavior",
				CounterEvidence:  []string{"calls can have effects and constant predicates may be intentional; no automatic fix"},
				Sites:            []Site{{Path: function.Path, Line: hit.Line, EndLine: hit.EndLine, ID: function.ID, Name: function.Name}},
			})
		}
	}
	return out
}
