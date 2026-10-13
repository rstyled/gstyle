package patterns

import "slices"

// Native analyses run once per function before matching. Their compact results
// let matching release canonical graph records instead of pinning a second corpus.
type semanticResults struct {
	findings []Candidate
	coverage []AnalysisCoverage
}

func (r *semanticResults) candidates(kind CandidateKind) []Candidate {
	var out []Candidate
	for _, candidate := range r.findings {
		if candidate.Kind == kind {
			out = append(out, candidate)
		}
	}
	return out
}

func nativeSemanticResults(facts []*MiningFacts) *semanticResults {
	var findings []Candidate
	for _, original := range facts {
		f := semanticFact(original)
		if f == nil || f.Pdg == nil {
			continue
		}
		findings = append(findings, semanticFunctionCandidates(f)...)
	}
	return &semanticResults{findings: findings, coverage: semanticCoverage(facts)}
}

func semanticFunctionCandidates(f *MiningFacts) []Candidate {
	facts := []*MiningFacts{f}
	return slices.Concat(
		TaintFlowCandidates(taintFlowsForFunction(f), facts),
		BarrierSliceCandidates(barrierSlicesForFunc(f), facts),
		ThinSliceCandidates(thinSlicesForFunc(f), facts),
		ChopCandidates(chopsForFunc(f), facts),
	)
}

func reportSemanticCoverage(facts []*MiningFacts, options Options) []AnalysisCoverage {
	if options.semantic != nil {
		return options.semantic.coverage
	}
	return semanticCoverage(facts)
}
