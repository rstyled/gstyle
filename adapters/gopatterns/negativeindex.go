package gopatterns

import (
	"fmt"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/pdg/query"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

// A missing lower bound is not evidence that a negative value reaches the access.
func negativeIndexCandidates(function FuncPdg, report runtimeguard.Report) []patterns.Candidate {
	var out []patterns.Candidate
	seen := make(map[pdg.Ref]bool)
	for _, site := range report.Accesses {
		if !uncoveredNegativeIndex(site) || seen[site.Symbol] {
			continue
		}
		node, present := query.At(function.Pdg.Nodes, site.Node)
		if !present {
			continue
		}
		seen[site.Symbol] = true
		out = append(out, negativeIndexCandidate(function, site, node.Line))
	}
	return out
}

func knownNegativeIndex(site runtimeguard.Access) bool {
	return site.Index && site.Value.HasUpper && site.Value.Upper < 0
}

func negativeIndexCandidate(function FuncPdg, site runtimeguard.Access, line uint32) patterns.Candidate {
	name := runtimeSymbolName(&function.Pdg, site.Symbol)
	return patterns.Candidate{
		Kind: patterns.NegativeIndex, ScoreMilli: 800,
		Observation:     fmt.Sprintf("%s is negative at this index (recorded upper bound %d)", name, site.Value.Upper),
		Inference:       "prevent negative values from reaching this index",
		CounterEvidence: []string{"based on recorded index bounds; no automatic fix"},
		Sites:           []patterns.Site{{Path: function.Path, Line: int(line), EndLine: int(line), ID: function.Name, Name: function.Name}},
	}
}

// A known-extent finding already explains this access. Report it once.
func uncoveredNegativeIndex(site runtimeguard.Access) bool {
	return knownNegativeIndex(site) && !literalIndexRisk(site)
}
