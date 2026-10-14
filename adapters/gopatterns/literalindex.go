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
			Observation:     "composite literal indexed by unchecked variable " + name,
			Inference:       "verify out-of-range values are rejected before the index is used; prefer a switch with default",
			CounterEvidence: []string{"intent unknown; no automatic fix — verify against the contract"},
			Sites:           []patterns.Site{{Path: function.Path, Line: line, EndLine: line, ID: function.Name, Name: function.Name}},
		})
	}
	return out
}

func literalIndexRisk(site runtimeguard.Access) bool {
	if !site.Literal || site.Constant {
		return false
	}
	return !validLiteralBounds(site.Value)
}

func validLiteralBounds(value runtimeguard.Value) bool {
	if !value.HasExtent || !value.HasLower || !value.HasUpper {
		return false
	}
	return value.Lower >= 0 && value.Upper < value.Extent
}
