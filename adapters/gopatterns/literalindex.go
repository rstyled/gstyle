package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

func literalIndexCandidates(function FuncPdg, report runtimeguard.Report) []patterns.Candidate {
	var out []patterns.Candidate
	for _, site := range report.Accesses {
		if !literalIndexRisk(site) {
			continue
		}
		name := runtimeSymbolName(&function.Pdg, site.Symbol)
		line := int(function.Pdg.Nodes[site.Node-1].Line)
		out = append(out, patterns.Candidate{
			Kind: patterns.LiteralIndex, ScoreMilli: 800,
			Observation:     "collection indexed out of bounds by " + name,
			Inference:       "prevent this out-of-range value from reaching the index",
			CounterEvidence: []string{"based on recorded index bounds and collection extent; no automatic fix"},
			Sites:           []patterns.Site{{Path: function.Path, Line: line, EndLine: line, ID: function.Name, Name: function.Name}},
		})
	}
	return out
}

// Unknown bounds do not establish an out-of-range access. Require the entire
// recorded range to lie below zero or at/above the collection extent.
func literalIndexRisk(site runtimeguard.Access) bool {
	if !site.Index || !site.Value.HasExtent {
		return false
	}
	value := site.Value
	if value.Extent == 0 {
		return true
	}
	if knownNegativeIndex(site) {
		return true
	}
	return value.HasLower && value.Lower >= value.Extent
}
