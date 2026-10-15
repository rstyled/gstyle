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
	result.LiteralIndices = nil
	for _, function := range result.Funcs {
		if !declarationPathSelected(root, filepath.Join(root, function.Path), paths) {
			continue
		}
		report, err := runtimeguard.Analyze(&function.Pdg)
		if err != nil {
			return fmt.Errorf("runtime guards for %s: %w", function.Name, err)
		}
		appendRuntimeCandidates(result, function, report)
	}
	return nil
}

func nilGuardCandidates(function FuncPdg, report runtimeguard.Report) []patterns.Candidate {
	var out []patterns.Candidate
	seen := make(map[pdg.Ref]bool)
	for _, site := range report.Accesses {
		if seen[site.Symbol] || !nilGuardRisk(&function.Pdg, site) {
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

// An unchanged entry receiver is a method-contract input, not a competing
// argument guard. Do not extend this exception to nil or replaced values.
func nilGuardRisk(graph *pdg.Graph, site runtimeguard.Access) bool {
	if site.Index || site.Value.NonNil || len(site.Guards) == 0 {
		return false
	}
	return !entryReceiverAccess(graph, site)
}

// runtimeguard.binding gives a stable entry pointer identity symbol*2. Writes
// preserve it only when the original value is copied back; joins, unknown call
// results and escaped bindings cannot establish this identity. Nil path facts
// still take precedence over identity.
func entryReceiverAccess(graph *pdg.Graph, site runtimeguard.Access) bool {
	if site.Value.Nil || site.Symbol == 0 {
		return false
	}
	if site.Value.Identity != uint64(site.Symbol)*2 {
		return false
	}
	return receiverSymbol(graph, site.Symbol)
}

// Normal-flow lowering can store receivers as parameter symbols. The function
// interface retains their declared role, independently of that storage choice.
func receiverSymbol(graph *pdg.Graph, symbol pdg.Ref) bool {
	if symbol == 0 {
		return false
	}
	for _, input := range graph.Function.Inputs {
		if input.Role == pdg.ReceiverRole && input.Symbol == symbol {
			return true
		}
	}
	return false
}

func runtimeSymbolName(graph *pdg.Graph, symbol pdg.Ref) string {
	if symbol == 0 {
		return "pointer expression"
	}
	return graph.Tables.Text(graph.Symbols[symbol-1].Name.Value)
}

func runtimeKindSuppressed(function FuncPdg, kind string) bool {
	for _, suppressed := range function.SuppressedKinds {
		if suppressed == kind {
			return true
		}
	}
	return false
}

func appendRuntimeCandidates(result *Extraction, function FuncPdg, report runtimeguard.Report) {
	if !runtimeKindSuppressed(function, "nil_guard") {
		result.NilGuards = append(result.NilGuards, nilGuardCandidates(function, report)...)
	}
	if !runtimeKindSuppressed(function, "negative_index") {
		result.NegativeIndices = append(result.NegativeIndices, negativeIndexCandidates(function, report)...)
	}
	if !runtimeKindSuppressed(function, "literal_index") {
		result.LiteralIndices = append(result.LiteralIndices, literalIndexCandidates(function, report)...)
	}
}
