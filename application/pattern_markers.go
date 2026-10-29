package application

import (
	"cmp"
	"context"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rstyled/gstyle/application/patterncheck"
	"github.com/rstyled/gstyle/domain/patterns"
)

// patternBackground is the line highlight for pattern findings. Distinct from
// the user-annotation bronze (44;34;14) and the selection slate (45;52;54).
const patternBackground = "\x1b[48;2;66;54;20m"

// PatternMarker is one visual marker for a pattern finding site.
type PatternMarker = FindingMarker

// patternsMsg carries background pattern-detection results.
type patternsMsg struct {
	markers    []PatternMarker
	err        error
	generation uint64
}

// loadPatternsCmd runs pattern detection in the background so TUI startup
// stays fast. Results arrive as patternsMsg.
func (m Model) loadPatternsCmd() tea.Cmd {
	paths := append([]string(nil), m.paths...)
	generation := m.patternGeneration
	return func() tea.Msg {
		report, err := patterncheck.GetReport(context.Background(), paths)
		if err != nil {
			return patternsMsg{err: err, generation: generation}
		}
		return patternsMsg{markers: patternMarkersFromReport(report), generation: generation}
	}
}

// selectedFilePath returns the selected file's path, or "" when none.
func (m Model) selectedFilePath() string {
	file, ok := m.selectedFile()
	if !ok {
		return ""
	}
	return file.Path
}

// togglePatterns flips pattern highlight visibility.
func (m Model) togglePatterns() Model {
	m.showPatterns = !m.showPatterns
	m.findingDetailOffset = 0
	if len(patternMarkersStartingAt(m.findingMarkers(), m.selectedFilePath(), m.activeFindingLine)) == 0 {
		m.activeFindingLine = 0
	}
	m.revision++
	return m
}

func (mk FindingMarker) covers(path string, line int) bool {
	if !patternPathMatches(mk.Path, path) {
		return false
	}
	end := mk.EndLine
	if end <= 0 {
		end = mk.Line
	}
	return mk.Line <= line && line <= end
}

// patternMarkersFromReport converts pattern candidates' sites into markers.
func patternMarkersFromReport(report *patterns.PatternsReport) []PatternMarker {
	if report == nil {
		return nil
	}
	var out []PatternMarker
	for i := range report.Candidates {
		c := &report.Candidates[i]
		kind := string(c.Kind)
		detail := &patterns.Candidate{
			Observation: c.Observation, Inference: c.Inference, PossibleRefactor: c.PossibleRefactor,
			Sites: c.Sites, CounterEvidence: c.CounterEvidence,
		}
		for _, s := range c.Sites {
			out = append(out, PatternMarker{
				Kind:      kind,
				Path:      s.Path,
				Line:      s.Line,
				EndLine:   s.EndLine,
				Candidate: detail,
			})
		}
	}
	slices.SortStableFunc(out, func(a, b PatternMarker) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Kind, b.Kind))
	})
	return out
}

// patternPathMatches reports whether a marker path and a file path refer to
// the same file, tolerating relative/absolute prefix differences.
func patternPathMatches(markerPath, filePath string) bool {
	if markerPath == filePath {
		return true
	}
	return pathHasSuffix(markerPath, filePath) || pathHasSuffix(filePath, markerPath)
}

func pathHasSuffix(path, suffix string) bool {
	return strings.HasSuffix(path, suffix) && strings.HasSuffix(strings.TrimSuffix(path, suffix), "/")
}

// patternMarkersAtLine returns markers covering path:line, deduplicated by kind.
func patternMarkersAtLine(markers []PatternMarker, path string, line int) []PatternMarker {
	var out []PatternMarker
	seen := map[string]bool{}
	for _, mk := range markers {
		if mk.covers(path, line) && !seen[mk.Kind] {
			seen[mk.Kind] = true
			out = append(out, mk)
		}
	}
	return out
}

// highlightPatternLine applies the pattern background to a rendered line when
// markers cover the line. Annotation and selection highlights are applied
// afterwards and take precedence.
func highlightPatternLine(rendered string, markers []PatternMarker) string {
	if len(markers) == 0 {
		return rendered
	}
	return withBackground(rendered, patternBackground)
}

func sourcePatternRows(markers []PatternMarker, width int) []string {
	if len(markers) == 0 {
		return nil
	}
	markers = uniquePatternMarkers(markers)
	slices.SortStableFunc(markers, func(a, b PatternMarker) int { return cmp.Compare(a.Kind, b.Kind) })
	prefix := muted.Render("      ╰─") + amber.Bold(true).Render("▲ ")
	var rows []string
	for _, marker := range markers {
		rows = append(rows, annotationMessageRows(marker.label(), prefix, width)...)
		for _, message := range marker.detailMessages() {
			rows = append(rows, annotationMessageRows(message, "          ", width)...)
		}
	}
	return rows
}

// patternMarkersStartingAt returns markers whose range starts at the line.
// Used for the label row, which shows once per finding, while the background
// highlight covers the whole range.
func patternMarkersStartingAt(markers []PatternMarker, path string, line int) []PatternMarker {
	var out []PatternMarker
	for _, mk := range markers {
		if !patternPathMatches(mk.Path, path) || mk.Line != line {
			continue
		}
		out = append(out, mk)
	}
	return uniquePatternMarkers(out)
}

// sourcePatternRowCount counts display rows added by pattern markers, for
// scroll math. Mirrors sourceAnnotationRowCount.
func sourcePatternRowCount(markers []PatternMarker, path string, start int, end int, width int) int {
	count := 0
	for line := start; line <= end; line++ {
		count += len(sourcePatternRows(patternMarkersStartingAt(markers, path, line), width))
	}
	return count
}
