package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
	"path/filepath"
	"strings"
)

// Derive findings only after the canonical graphs are built. Cached package
// extraction uses this same entry point and never retains compiler objects.
func populateRuntimeGuards(result *Extraction, root string, paths []string) error {
	result.NilGuards = nil
	result.NegativeIndices = nil
	for _, function := range result.Funcs {
		if !declarationPathSelected(root, filepath.Join(root, function.Path), paths) {
			continue
		}
		report, err := runtimeguard.Analyze(&function.Pdg)
		if err != nil {
			return fmt.Errorf("runtime guards for %s: %w", function.Name, err)
		}
		result.NilGuards = append(result.NilGuards, nilGuardCandidates(function, report)...)
		result.NegativeIndices = append(result.NegativeIndices, negativeIndexCandidates(function, report)...)
	}
	return nil
}

func nilGuardCandidates(function FuncPdg, report runtimeguard.Report) []patterns.Candidate {
	var out []patterns.Candidate
	seen := make(map[pdg.Ref]bool)
	for _, site := range report.Accesses {
		if site.Index || site.Value.NonNil || len(site.Guards) == 0 || seen[site.Symbol] {
			continue
		}
		seen[site.Symbol] = true
		out = append(out, buildNilGuardCandidate(function, site))
	}
	return out
}

func buildNilGuardCandidate(function FuncPdg, site runtimeguard.Access) patterns.Candidate {
	var names []string
	for _, symbol := range site.Guards {
		names = append(names, runtimeSymbolName(&function.Pdg, symbol))
	}
	name := runtimeSymbolName(&function.Pdg, site.Symbol)
	line := int(function.Pdg.Nodes[site.Node-1].Line)
	return patterns.Candidate{
		Kind: patterns.NilGuard, ScoreMilli: 800,
		Observation:     "nil guard checks " + strings.Join(names, ", ") + " while " + name + " is dereferenced without a guard",
		Inference:       "verify the guard targets the dereferenced pointer; the checked variable may be the wrong one",
		CounterEvidence: []string{"intent unknown; no automatic fix — verify against the contract"},
		Sites:           []patterns.Site{{Path: function.Path, Line: line, EndLine: line, ID: function.Name, Name: function.Name}},
	}
}

func runtimeSymbolName(graph *pdg.Graph, symbol pdg.Ref) string {
	if symbol == 0 {
		return "pointer expression"
	}
	return graph.Tables.Text(graph.Symbols[symbol-1].Name.Value)
}
