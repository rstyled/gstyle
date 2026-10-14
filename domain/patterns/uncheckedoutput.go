package patterns

import "github.com/rstyled/gstyle/domain/patterns/outputcheck"

func uncheckedOutputCandidates(facts []*MiningFacts) []Candidate {
	var out []Candidate
	for _, function := range facts {
		context := outputcheck.Function{Path: function.Path, ID: function.ID, Name: function.Name}
		for _, finding := range outputcheck.Report(context, function.ErrorCheckSites, function.SuppressedKinds) {
			out = append(out, Candidate{
				Kind: CandidateKind(finding.Kind), ScoreMilli: finding.ScoreMilli,
				Observation:     finding.Observation,
				Inference:       finding.Inference,
				CounterEvidence: []string{finding.CounterEvidence},
				Sites:           []Site{{Path: finding.Path, Line: finding.Line, EndLine: finding.EndLine, ID: finding.ID, Name: finding.Name}},
			})
		}
	}
	return out
}
