package patterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

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

func nativeSemanticResults(facts []*MiningFacts, graphs []*pdg.Graph) *semanticResults {
	catalog, err := semanticCatalog(facts, graphs)
	var flows []TaintFlow
	if err == nil {
		flows, err = QueryTaintFlows(catalog)
	}
	findings := TaintFlowCandidates(flows, catalog)
	for _, original := range facts {
		f := semanticFact(original)
		if f == nil || f.Pdg == nil {
			continue
		}
		findings = append(findings, semanticFunctionCandidates(f)...)
	}
	return &semanticResults{findings: findings, coverage: taintQueryCoverage(catalog, err)}
}

func semanticFunctionCandidates(f *MiningFacts) []Candidate {
	facts := []*MiningFacts{f}
	return slices.Concat(
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

func taintQueryCoverage(facts []*MiningFacts, err error) []AnalysisCoverage {
	coverage := semanticCoverage(facts)
	if err == nil {
		return coverage
	}
	for i := range coverage {
		row := &coverage[i]
		if row.Kind != TaintFlowKind {
			continue
		}
		row.Unknown += row.Complete + row.Partial
		row.Complete, row.Partial = 0, 0
		row.Reason = err.Error()
	}
	return coverage
}
