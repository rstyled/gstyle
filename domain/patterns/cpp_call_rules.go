package patterns

import "slices"

func cppCallRules(facts []*MiningFacts) []Candidate {
	var closed []*MiningFacts
	for _, fact := range facts {
		if !slices.ContainsFunc(fact.Pdg.Nodes, func(node PdgNode) bool { return node.Kind == Call && node.CalleeID == "" }) {
			closed = append(closed, fact)
		}
	}
	sets := BuildCallSets(closed)
	result := MinedRuleCandidates(FindViolations(sets, MineRules(sets)), closed)
	result = append(result, AlattinRuleCandidates(FindAlternativeViolations(sets, MineDisjunctiveRules(sets)), closed)...)
	for i := range result {
		result[i].PossibleRefactor = "review this call association against the C++ API contract and error paths"
		result[i].CounterEvidence = append(result[i].CounterEvidence, "a mined association does not prove that a call is required; functions with opaque call targets are excluded")
	}
	return result
}
