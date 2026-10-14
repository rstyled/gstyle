package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
	"github.com/rstyled/gstyle/domain/runtimeguard/interval"
)

func negativeIndexCandidates(function FuncPdg, report runtimeguard.Report) []patterns.Candidate {
	var out []patterns.Candidate
	var accesses []interval.Access
	for _, site := range report.Accesses {
		accesses = append(accesses, interval.Access{Node: uint32(site.Node), Symbol: uint32(site.Symbol), Index: site.Index, UpperChecked: site.Value.UpperChecked, Bounds: site.Value.Bounds()})
	}
	for _, site := range interval.Select(accesses) {
		name := runtimeSymbolName(&function.Pdg, pdg.Ref(site.Symbol))
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
