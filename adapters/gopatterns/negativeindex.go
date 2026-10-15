package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/pdg/query"
	"github.com/rstyled/gstyle/domain/runtimeguard"
	"github.com/rstyled/gstyle/domain/runtimeguard/interval"
)

func negativeIndexCandidates(function FuncPdg, report runtimeguard.Report) []patterns.Candidate {
	var out []patterns.Candidate
	var accesses []interval.Access
	for _, site := range report.Accesses {
		if _, valid := query.ReferenceIndex(site.Node, len(function.Pdg.Nodes)); !valid {
			continue
		}
		accesses = append(accesses, interval.Access{Node: uint32(site.Node), Symbol: uint32(site.Symbol), Index: site.Index, UpperChecked: site.Value.UpperChecked, Bounds: site.Value.Bounds()})
	}
	for _, site := range interval.Select(accesses) {
		node, present := query.At(function.Pdg.Nodes, pdg.Ref(site.Node))
		if !present {
			continue
		}
		name := runtimeSymbolName(&function.Pdg, pdg.Ref(site.Symbol))
		finding := interval.Describe(function.Path, function.Name, name, node.Line)
		out = append(out, patterns.Candidate{
			Kind: patterns.CandidateKind(finding.Kind), ScoreMilli: finding.ScoreMilli,
			Observation:     finding.Observation,
			Inference:       finding.Inference,
			CounterEvidence: []string{finding.CounterEvidence},
			Sites:           []patterns.Site{{Path: finding.Path, Line: finding.Line, EndLine: finding.EndLine, ID: finding.ID, Name: finding.Name}},
		})
	}
	return out
}
