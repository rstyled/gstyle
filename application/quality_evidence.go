package application

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/quality"
)

type qualityEvidence struct {
	line    int
	message string
}

func qualityDiagnosticMarkers(path string, function domain.Function, diagnostic quality.Diagnostic) []PatternMarker {
	if diagnostic.RuleID == "mutated_targets" && len(diagnostic.Mutations) == 0 {
		diagnostic.Mutations = function.Quality.MutationEvents
	}
	summary := qualityFindingMarker(path, function, diagnostic)
	groups := groupedQualityEvidence(path, function, summary)
	summary.Details = &findingDetails{messages: append(summary.qualityDetailMessages(), distinctEvidenceLines(groups[summary.Line])...)}
	markers := []PatternMarker{summary}
	for _, line := range slices.Sorted(maps.Keys(groups)) {
		if line == summary.Line {
			continue
		}
		markers = append(markers, FindingMarker{
			Kind: summary.Kind, Path: summary.Path, Line: line, EndLine: line, Scope: "source evidence",
			Diagnostic: summary.Diagnostic, Details: &findingDetails{messages: distinctEvidenceLines(groups[line])},
		})
	}
	return markers
}

func groupedQualityEvidence(path string, function domain.Function, summary FindingMarker) map[int][]string {
	groups := make(map[int][]string)
	for _, evidence := range diagnosticEvidence(*summary.Diagnostic) {
		line := evidence.line
		if !patternPathMatches(path, summary.Path) || !functionHasSourceLine(function, line) {
			line = summary.Line
			evidence.message = "Source location unavailable: " + evidence.message
		} else {
			evidence.message = strings.TrimPrefix(evidence.message, fmt.Sprintf("L%d ", line))
		}
		groups[line] = append(groups[line], evidence.message)
	}
	return groups
}

func diagnosticEvidence(diagnostic quality.Diagnostic) []qualityEvidence {
	var records []qualityEvidence
	unknowns := make(map[int][]quality.Effect)
	for _, effect := range diagnostic.Effects {
		if effect.Kind == quality.UnknownEffect {
			unknowns[effect.Line] = append(unknowns[effect.Line], effect)
			continue
		}
		records = append(records, qualityEvidence{line: effect.Line, message: fmt.Sprintf("L%d %s: %s", effect.Line, effect.Kind, effect.Detail)})
	}
	for _, line := range slices.Sorted(maps.Keys(unknowns)) {
		records = append(records, qualityEvidence{line: line, message: inlineUnknownEvidence(line, unknowns[line])})
	}
	for _, mutation := range diagnostic.Mutations {
		records = append(records, qualityEvidence{line: mutation.Line, message: mutationQualityLine(mutation)})
	}
	return records
}

func inlineUnknownEvidence(line int, effects []quality.Effect) string {
	if len(effects) == 1 {
		return fmt.Sprintf("L%d unknown: %s", line, effects[0].Detail)
	}
	if effects[0].Site != 0 && !slices.ContainsFunc(effects, func(effect quality.Effect) bool { return effect != effects[0] }) {
		return fmt.Sprintf("L%d unknown: %s (%d records; e shows full evidence)", line, effects[0].Detail, len(effects))
	}
	return fmt.Sprintf("L%d unknown: %d unresolved effect records; e shows full evidence", line, len(effects))
}
