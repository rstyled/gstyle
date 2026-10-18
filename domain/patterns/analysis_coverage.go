package patterns

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/pdg/query"
)

// Coverage is analysis completeness, not a probability that a finding is a bug.
type AnalysisEvidence struct {
	Related     []*AnalysisEvidence `json:"related_functions,omitempty"`
	FunctionID  string              `json:"function_id,omitempty"`
	Conformance string              `json:"graph_conformance,omitempty"`
	Status      string              `json:"status"`
	NodeIDs     []string            `json:"node_ids,omitempty"`
	Missing     []string            `json:"missing_or_approximate,omitempty"`
}

type AnalysisCoverage struct {
	Reason      string        `json:"reason,omitempty"`
	Kind        CandidateKind `json:"analysis"`
	Complete    int           `json:"complete"`
	Partial     int           `json:"partial"`
	Unknown     int           `json:"not_evaluated"`
	Unavailable int           `json:"unavailable"`
}

var semanticAnalysisKinds = []CandidateKind{TaintFlowKind, BarrierSliceKind, ThinSliceKind, ChopKind}

var sourceFeatureCapabilities = map[string]string{
	"*ast.ForStmt": "loop_carried_dependencies", "*ast.RangeStmt": "loop_carried_dependencies",
	"*ast.FuncLit": "closures", "*ast.GoStmt": "concurrency", "*ast.SelectStmt": "concurrency", "*ast.SendStmt": "concurrency",
}

func analysisEvidence(f *MiningFacts, kind CandidateKind, nodes ...string) *AnalysisEvidence {
	ids := nonemptyStrings(nodes)
	if f == nil {
		return &AnalysisEvidence{Status: "not_evaluated", NodeIDs: ids}
	}
	if f.Canonical == nil {
		return &AnalysisEvidence{FunctionID: f.ID, Status: "not_evaluated", NodeIDs: ids}
	}
	graph := f.Canonical
	gaps := analysisGaps(graph, kind)
	status := "complete"
	if len(gaps) != 0 {
		status = "partial"
	}
	return &AnalysisEvidence{FunctionID: f.ID, Conformance: graphConformanceName(graph.Conformance.Status), Status: status, NodeIDs: ids, Missing: gaps}
}

func nonemptyStrings(nodes []string) []string {
	var ids []string
	for _, node := range nodes {
		if node != "" {
			ids = append(ids, node)
		}
	}
	return ids
}

func analysisGaps(graph *pdg.Graph, kind CandidateKind) []string {
	return slices.Concat(capabilityGaps(graph, analysisRequirements(graph, kind)), dependencyEvidenceGaps(graph, kind), taintSummaryLimitations(kind))
}

func capabilityGaps(graph *pdg.Graph, required []string) []string {
	var gaps []string
	for _, name := range required {
		status := graphCapability(graph, name)
		if !supportedCapability(status) {
			gaps = append(gaps, name+": "+coverageStatusName(status))
		}
	}
	return gaps
}

func dependencyEvidenceGaps(graph *pdg.Graph, kind CandidateKind) []string {
	if supportedEdges(graph, kind) {
		return nil
	}
	return []string{"dependency edge evidence is missing or approximate"}
}

func graphCapability(graph *pdg.Graph, name string) pdg.Status {
	if graph == nil || graph.Tables == nil {
		return pdg.Unknown
	}
	for _, capability := range graph.Capabilities {
		if graph.Tables.Text(capability.Name) == name {
			return evidenceStatus(graph, capability.Evidence)
		}
	}
	return pdg.Unknown
}

func evidenceStatus(graph *pdg.Graph, ref pdg.Ref) pdg.Status {
	if graph == nil || graph.Tables == nil {
		return pdg.Unknown
	}
	return query.EvidenceStatus(graph.Tables.Evidence, ref)
}

func supportedCapability(status pdg.Status) bool {
	return query.CoverageSupported(status)
}

func supportedEdges(graph *pdg.Graph, kind CandidateKind) bool {
	if graph == nil || graph.Tables == nil {
		return false
	}
	for _, edge := range graph.Edges {
		if !semanticDependency(edge.Kind, kind) {
			continue
		}
		if !query.DependencyAdmitted(graph.Tables.Evidence, edge.Evidence) {
			return false
		}
	}
	return true
}

func coverageStatusName(status pdg.Status) string {
	switch status {
	case pdg.Approximated:
		return "approximated"
	case pdg.Unsupported:
		return "unsupported"
	default:
		return "unknown"
	}
}

func semanticCoverage(facts []*MiningFacts) []AnalysisCoverage {
	coverage := make([]AnalysisCoverage, len(semanticAnalysisKinds))
	for i, kind := range semanticAnalysisKinds {
		counts := make(map[string]int, 4)
		for _, f := range facts {
			counts[functionCoverageStatus(f, kind)]++
		}
		coverage[i] = AnalysisCoverage{Kind: kind, Complete: counts["complete"], Partial: counts["partial"], Unknown: counts["not_evaluated"], Unavailable: counts["unavailable"]}
	}
	return coverage
}

func functionCoverageStatus(f *MiningFacts, kind CandidateKind) string {
	if f == nil || f.Pdg == nil {
		return "unavailable"
	}
	if f.Canonical == nil {
		return "not_evaluated"
	}
	if !supportedCapability(graphCapability(f.Canonical, "node_extraction")) {
		return "partial"
	}
	if len(analysisGaps(f.Canonical, kind)) != 0 {
		return "partial"
	}
	return "complete"
}

func coverageText(coverage []AnalysisCoverage) string {
	if len(coverage) == 0 {
		return ""
	}
	lines := []string{"Dependency analysis coverage (no findings does not prove clean flow):"}
	for _, row := range coverage {
		lines = append(lines, fmt.Sprintf("  %s: %d complete, %d partial, %d not evaluated, %d unavailable", row.Kind, row.Complete, row.Partial, row.Unknown, row.Unavailable))
		if row.Reason != "" {
			lines = append(lines, "    "+row.Reason)
		}
	}
	return strings.Join(lines, "\n")
}

func analysisRequirements(graph *pdg.Graph, kind CandidateKind) []string {
	return slices.Concat([]string{"node_extraction", "scalar_reaching_definitions", "returns"}, specificRequirements(kind), memoryRequirements(graph), sourceFeatureRequirements(graph))
}

func specificRequirements(kind CandidateKind) []string {
	switch kind {
	case BarrierSliceKind, ChopKind:
		return []string{"control_dependencies"}
	case TaintFlowKind:
		return []string{"calls"}
	default:
		return nil
	}
}

func memoryRequirements(graph *pdg.Graph) []string {
	if len(graph.Locations) == 0 {
		return nil
	}
	return []string{"memory_aliases"}
}

func semanticDependency(edge pdg.EdgeKind, kind CandidateKind) bool {
	if edge == pdg.Execution {
		return false
	}
	if edge == pdg.Data {
		return true
	}
	return kind == BarrierSliceKind || kind == ChopKind
}

func graphConformanceName(status pdg.ConformanceStatus) string {
	switch status {
	case pdg.Conformant:
		return "conformant"
	case pdg.NonConformant:
		return "non_conformant"
	default:
		return "not_evaluated"
	}
}

// HasIncompleteAnalysis keeps next/score output from treating absent findings
// as proof that unsupported dependency analysis passed.
func HasIncompleteAnalysis(coverage []AnalysisCoverage) bool {
	if len(coverage) == 0 {
		return true
	}
	for _, row := range coverage {
		if row.Partial != 0 || row.Unknown != 0 || row.Unavailable != 0 {
			return true
		}
	}
	return false
}

func sourceFeatureRequirements(graph *pdg.Graph) []string {
	seen := make(map[string]bool)
	var required []string
	for _, node := range graph.Nodes {
		capability := sourceFeatureCapabilities[graph.Tables.Text(node.OriginalKind.Value)]
		if capability == "" || seen[capability] {
			continue
		}
		seen[capability] = true
		required = append(required, capability)
	}
	return required
}

func taintSummaryLimitations(kind CandidateKind) []string {
	if kind != TaintFlowKind {
		return nil
	}
	return []string{"taint summaries are partial: validation, dynamic targets, collection contents and exceptional field effects are not proved"}
}
