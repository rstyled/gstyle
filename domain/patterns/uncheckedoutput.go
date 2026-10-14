package patterns

func uncheckedOutputCandidates(facts []*MiningFacts) []Candidate {
	var out []Candidate
	for _, function := range facts {
		if isSuppressed(function.SuppressedKinds, UncheckedOutput) {
			continue
		}
		for _, site := range function.ErrorCheckSites {
			if !site.DiscardedOutput {
				continue
			}
			out = append(out, Candidate{
				Kind: UncheckedOutput, ScoreMilli: 800,
				Observation:     "formatted output error is discarded: " + site.Callee,
				Inference:       "a failed write can be reported as success; propagate or explicitly handle the output error",
				CounterEvidence: []string{"best-effort output may deliberately ignore failures; no automatic fix"},
				Sites:           []Site{{Path: function.Path, Line: site.Line, EndLine: site.Line, ID: function.ID, Name: function.Name}},
			})
		}
	}
	return out
}
