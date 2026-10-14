package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

func negativeIndexCandidates(function FuncPdg, report runtimeguard.Report) []patterns.Candidate {
	var out []patterns.Candidate
	seen := make(map[pdg.Ref]bool)
	for _, site := range report.Accesses {
		if !negativeIndexRisk(site) || seen[site.Symbol] {
			continue
		}
		seen[site.Symbol] = true
		name := runtimeSymbolName(&function.Pdg, site.Symbol)
		line := int(function.Pdg.Nodes[site.Node-1].Line)
		out = append(out, patterns.Candidate{
			Kind: patterns.NegativeIndex, ScoreMilli: 800,
			Observation:     "upper-bound check on " + name + " has no lower-bound check; " + name + " indexes a collection",
			Inference:       "verify negative indices are rejected or impossible before the index is used",
			CounterEvidence: []string{"intent unknown; no automatic fix — verify against the contract"},
			Sites:           []patterns.Site{{Path: function.Path, Line: line, EndLine: line, ID: function.Name, Name: function.Name}},
		})
	}
	return out
}

func negativeIndexRisk(site runtimeguard.Access) bool {
	if !site.Index || !site.Value.UpperChecked {
		return false
	}
	return !site.Value.HasLower || site.Value.Lower < 0
}
