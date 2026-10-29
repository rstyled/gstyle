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
	for _, effect := range diagnostic.Effects {
		records = append(records, qualityEvidence{line: effect.Line, message: fmt.Sprintf("L%d %s: %s", effect.Line, effect.Kind, effect.Detail)})
	}
	for _, mutation := range diagnostic.Mutations {
		records = append(records, qualityEvidence{line: mutation.Line, message: mutationQualityLine(mutation)})
	}
	return records
}
