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
	Candidate  *patterns.Candidate
	Diagnostic *quality.Diagnostic
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
	path := m.selectedFilePath()
	var markers []PatternMarker
	for _, marker := range m.findingMarkers() {
		if patternPathMatches(marker.Path, path) {
			markers = append(markers, marker)
		}
	}
	return markers
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
			line := diagnostic.Line
			if !functionHasSourceLine(function, line) {
				line = function.Line
			}
			markers = append(markers, PatternMarker{
				Kind: diagnostic.RuleID, Path: file.Path, Line: line, EndLine: line, Diagnostic: &diagnostic,
			})
		}
	}
	return markers
}

func qualityDetailMessages(diagnostic *quality.Diagnostic) []string {
	messages := []string{
		diagnostic.Message,
		fmt.Sprintf("At: %s:%d", diagnostic.Path, diagnostic.Line),
		fmt.Sprintf("Actual: %d · limit: %d", diagnostic.Actual, diagnostic.Limit),
	}
	messages = append(messages, effectLines(diagnostic.Effects)...)
	return append(messages, mutationQualityLines(diagnostic.Mutations)[2:]...)
}
