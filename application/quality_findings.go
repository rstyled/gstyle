package application

import (
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
)

type FindingMarker struct {
	Kind       string
	Path       string
	Line       int
	EndLine    int
	Scope      string
	Candidate  *patterns.Candidate
	Diagnostic *quality.Diagnostic
	Details    *findingDetails
}

type findingDetails struct {
	messages []string
}

func (marker FindingMarker) symbol() string {
	if marker.Diagnostic != nil {
		return "■ "
	}
	return "▲ "
}

func (marker FindingMarker) label() string {
	if marker.Diagnostic != nil {
		return "quality: " + marker.Kind
	}
	return "pattern: " + marker.Kind
}

func (m Model) findingMarkers() []PatternMarker {
	qualityMarkers := m.qualityMarkers
	if qualityMarkers == nil {
		qualityMarkers = qualityMarkersFromReport(m.report)
	}
	if !m.showPatterns {
		return qualityMarkers
	}
	return slices.Concat(m.patternMarkers, qualityMarkers)
}

func (m Model) visibleFindingMarkers() []PatternMarker {
	m = m.ensureFindingIndex()
	path := m.selectedFilePath()
	qualityMarkers := m.findingLookup.quality[path].markers
	if !m.showPatterns {
		return qualityMarkers
	}
	return slices.Concat(m.findingLookup.patterns[path].markers, qualityMarkers)
}

func qualityMarkersFromReport(report domain.Report) []PatternMarker {
	markers := make([]PatternMarker, 0)
	for _, file := range report.Files {
		markers = append(markers, fileQualityMarkers(file)...)
	}
	return markers
}

func fileQualityMarkers(file domain.File) []PatternMarker {
	var markers []PatternMarker
	for _, function := range file.Functions {
		if function.Quality == nil {
			continue
		}
		for _, diagnostic := range function.Quality.Diagnostics {
			markers = append(markers, qualityDiagnosticMarkers(file.Path, function, diagnostic)...)
		}
	}
	return markers
}

func qualityFindingMarker(path string, function domain.Function, diagnostic quality.Diagnostic) FindingMarker {
	if diagnostic.Path == "" {
		diagnostic.Path = path
	}
	line, scope := qualityMarkerLocation(path, function, diagnostic)
	return FindingMarker{Kind: diagnostic.RuleID, Path: diagnostic.Path, Line: line, EndLine: line, Scope: scope, Diagnostic: &diagnostic}
}

func qualityMarkerLocation(path string, function domain.Function, diagnostic quality.Diagnostic) (int, string) {
	if !patternPathMatches(path, diagnostic.Path) {
		return diagnostic.Line, "recorded source location"
	}
	if line := qualityMutationLine(function, diagnostic.Mutations); diagnostic.RuleID == "mutation_per_target" && line > 0 {
		return line, "mutation target"
	}
	if !functionHasSourceLine(function, diagnostic.Line) {
		return function.Line, fmt.Sprintf("recorded L%d is outside the available function source", diagnostic.Line)
	}
	if diagnostic.Line == function.Line {
		return function.Line, "whole function"
	}
	return diagnostic.Line, "source location"
}

func qualityMutationLine(function domain.Function, mutations []quality.Mutation) int {
	end := function.Line + sourceLineCount(function.Source)
	line := end
	for _, mutation := range mutations {
		if functionHasSourceLine(function, mutation.Line) {
			line = min(line, mutation.Line)
		}
	}
	if line == end {
		return 0
	}
	return line
}

func (marker FindingMarker) qualityDetailMessages() []string {
	diagnostic := marker.Diagnostic
	messages := []string{
		diagnostic.Message,
		fmt.Sprintf("Reported: %s:%d", diagnostic.Path, diagnostic.Line),
		fmt.Sprintf("Actual: %d · limit: %d", diagnostic.Actual, diagnostic.Limit),
	}
	if marker.Scope != "" {
		messages = append(messages, "Scope: "+marker.Scope)
	}
	if slices.ContainsFunc(diagnostic.Effects, func(effect quality.Effect) bool { return effect.Kind == quality.UnknownEffect }) {
		messages = append(messages, "Unknown effects are estimates, not confirmed side effects.")
	}
	return messages
}

func distinctEvidenceLines(evidence []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, line := range evidence {
		if !seen[line] {
			seen[line] = true
			unique = append(unique, line)
		}
	}
	return unique
}
