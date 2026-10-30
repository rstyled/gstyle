package patterns

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

type DependencyMeasurement struct {
	Kind        CandidateKind     `json:"analysis"`
	Site        Site              `json:"site"`
	SeedNode    string            `json:"seed_node_id,omitempty"`
	GraphNodes  int               `json:"graph_nodes"`
	Calls       int               `json:"function_calls"`
	NodeKinds   map[string]int    `json:"node_kinds"`
	SourceLines []int             `json:"source_lines"`
	Analysis    *AnalysisEvidence `json:"analysis_evidence,omitempty"`
}

func dependencyMeasurements(f *MiningFacts, kind CandidateKind) []DependencyMeasurement {
	return dependencyMeasurementsWithLimits(f, kind, DefaultConfig().Measurements)
}

func dependencyMeasurementsWithLimits(f *MiningFacts, kind CandidateKind, limits MeasurementLimits) []DependencyMeasurement {
	result := collectDependencyMeasurements(f, kind, limits)
	return slices.DeleteFunc(result, func(measurement DependencyMeasurement) bool {
		return measurement.Calls < limits.MinFunctionCalls || len(measurement.SourceLines) < limits.MinSourceLines
	})
}

func collectDependencyMeasurements(f *MiningFacts, kind CandidateKind, limits MeasurementLimits) []DependencyMeasurement {
	switch kind {
	case ThinSliceKind, BarrierSliceKind:
		return returnMeasurements(f, kind, limits)
	case ChopKind:
		chop := paramToReturnChop(f.Pdg)
		if len(chop) < limits.ChopMinNodes {
			return nil
		}
		return []DependencyMeasurement{measureDependencies(f, kind, sortedNodeSet(chop), -1)}
	case DependenceCluster:
		var result []DependencyMeasurement
		for _, cluster := range findDepClustersWithLimits([]*MiningFacts{f}, limits) {
			result = append(result, measureDependencies(f, kind, cluster.Nodes, -1))
		}
		return result
	}
	return nil
}

func sortedNodeSet(nodes map[int]bool) []int {
	var result []int
	for node := range nodes {
		result = append(result, node)
	}
	slices.Sort(result)
	return result
}

func returnMeasurements(f *MiningFacts, kind CandidateKind, limits MeasurementLimits) []DependencyMeasurement {
	thin := kind == ThinSliceKind
	traversal := newReturnTraversal(f.Pdg, thin)
	blocked, minimum := returnMeasurementPolicy(f.Pdg, thin, limits)
	var result []DependencyMeasurement
	for i, node := range f.Pdg.Nodes {
		if node.Kind != Return {
			continue
		}
		if traversal.size(i, blocked) >= minimum {
			result = append(result, measureDependencies(f, kind, traversal.queue, i))
		}
	}
	return result
}

func returnMeasurementPolicy(graph *MiningGraph, thin bool, limits MeasurementLimits) ([]bool, int) {
	blocked := make([]bool, len(graph.Nodes))
	if thin {
		return blocked, limits.ThinSliceMinNodes
	}
	for i, node := range graph.Nodes {
		blocked[i] = node.Kind == Try
	}
	return blocked, limits.BarrierSliceMinNodes
}

func measureDependencies(f *MiningFacts, kind CandidateKind, nodes []int, seed int) DependencyMeasurement {
	counts, lines := dependencyNodeCounts(f, nodes)
	site := Site{ID: f.ID, Name: f.Name, Path: f.Path, Line: f.Line, EndLine: f.EndLine}
	if seed >= 0 {
		site.Line = f.Pdg.Nodes[seed].Line
	}
	return DependencyMeasurement{Kind: kind, Site: site, SeedNode: canonicalNodeID(f.Canonical, seed), GraphNodes: len(nodes), Calls: counts["call"], NodeKinds: counts, SourceLines: lines}
}

func dependencyNodeCounts(f *MiningFacts, nodes []int) (map[string]int, []int) {
	counts := map[string]int{}
	lines := map[int]bool{}
	for _, index := range nodes {
		counts[dependencyNodeKind(f, index)]++
		if line := f.Pdg.Nodes[index].Line; line > 0 {
			lines[line] = true
		}
	}
	return counts, sortedNodeSet(lines)
}

func dependencyNodeKind(f *MiningFacts, index int) string {
	if f.Canonical == nil {
		return string(f.Pdg.Nodes[index].Kind)
	}
	node := f.Canonical.Nodes[index]
	if f.Canonical.Tables.Text(node.OriginalKind.Value) == "LITERAL" || f.Pdg.Nodes[index].Kind == Lit {
		return "literal"
	}
	return map[pdg.Category]string{pdg.Entry: "entry", pdg.Exit: "exit", pdg.FormalInput: "parameter", pdg.FormalOutput: "formal_output", pdg.Declaration: "declaration", pdg.Assignment: "assignment", pdg.Read: "read", pdg.Computation: "computation", pdg.Call: "call", pdg.Control: "control", pdg.Return: "return", pdg.Merge: "merge", pdg.MemorySummary: "memory_summary", pdg.Other: "other"}[node.Category]
}

func measurementText(measurements []DependencyMeasurement) string {
	if len(measurements) == 0 {
		return ""
	}
	counts := map[CandidateKind]int{}
	for _, measurement := range measurements {
		counts[measurement.Kind]++
	}
	lines := []string{fmt.Sprintf("%d dependency measurements (context only; no refactor finding from size alone)", len(measurements))}
	for _, kind := range FixKindOrder {
		if counts[kind] != 0 {
			lines = append(lines, fmt.Sprintf("  %s: %d", kind, counts[kind]))
		}
	}
	return strings.Join(append(lines, "JSON includes graph-node counts, function-call counts, node kinds, and source lines."), "\n")
}
