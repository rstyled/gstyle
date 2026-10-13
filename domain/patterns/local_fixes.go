package patterns

// RunFixCandidates avoids graph comparison for source-local fix families.
// These consume the same extraction facts and builders as the full report.
func RunFixCandidates(facts []*FuncFacts, options Options, kind CandidateKind) []Candidate {
	switch kind {
	case GuardClause:
		return guardCandidates(localGuardFacts(facts))
	case OffsetFallback:
		return options.OffsetFallbacks
	case DeclarationOrder:
		return options.DeclarationOrders
	default:
		return Run(facts, options).Candidates
	}
}

func localGuardFacts(facts []*FuncFacts) []*MiningFacts {
	var result []*MiningFacts
	for _, fact := range facts {
		if fact == nil {
			continue
		}
		result = append(result, &MiningFacts{ID: fact.ID, Name: fact.Name, Path: fact.Path, Line: fact.Line, EndLine: fact.EndLine, GuardClauses: fact.GuardClauses})
	}
	return result
}
