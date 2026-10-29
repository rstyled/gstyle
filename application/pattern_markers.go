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
	findings   *findingIndex
}

// loadPatternsCmd runs pattern detection in the background so TUI startup
// stays fast. Results arrive as patternsMsg.
func (m Model) loadPatternsCmd() tea.Cmd {
	paths := append([]string(nil), m.paths...)
	generation := m.patternGeneration
	load := func() (*patterns.PatternsReport, error) { return loadPatternReport(context.Background(), m.analyzer, paths) }
	return func() tea.Msg {
		report, err := load()
		if err != nil {
			return patternsMsg{err: err, generation: generation}
		}
		indexed := m
		indexed.patternMarkers = patternMarkersFromReport(report)
		indexed = indexed.indexFindings()
		return patternsMsg{markers: indexed.patternMarkers, generation: generation, findings: indexed.findingLookup}
	}
}

func loadPatternReport(ctx context.Context, analyzer Analyzer, paths []string) (*patterns.PatternsReport, error) {
	if frontend, ok := analyzer.(interface {
		Patterns(context.Context, []string) (*patterns.PatternsReport, error)
	}); ok {
		return frontend.Patterns(ctx, paths)
	}
	return patterncheck.GetReport(ctx, paths)
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
	if len(m.findingsStartingAt(m.activeFindingLine)) == 0 {
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
			marker := PatternMarker{
				Kind:      kind,
				Path:      s.Path,
				Line:      s.Line,
				EndLine:   s.EndLine,
				Candidate: detail,
			}
			marker.Details = &findingDetails{messages: marker.detailMessages()}
			out = append(out, marker)
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

func sourcePatternRows(markers []PatternMarker, width int) []string {
	if len(markers) == 0 {
		return nil
	}
	markers = uniquePatternMarkers(markers)
	slices.SortStableFunc(markers, func(a, b PatternMarker) int { return cmp.Compare(a.Kind, b.Kind) })
	var rows []string
	for _, marker := range markers {
		prefix := muted.Render("      ╰─") + amber.Bold(true).Render(marker.symbol())
		if marker.Scope != "source evidence" {
			rows = append(rows, annotationMessageRows(marker.label(), prefix, width)...)
			prefix = "          "
		}
		for _, message := range marker.detailMessages() {
			rows = append(rows, annotationMessageRows(message, prefix, width)...)
		}
	}
	return rows
}
