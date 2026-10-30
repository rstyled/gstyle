package application

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
)

func TestFindingIndexChangesWithAnalysisNotNavigation(t *testing.T) {
	m := qualityModel()
	m.showPatterns = true
	m = m.applyReport(m.report)
	qualityIndex := m.findingLookup
	candidate := &patterns.Candidate{Observation: "shared finding"}
	m = m.withPatterns(patternsMsg{markers: []PatternMarker{
		{Kind: "ccgraph_clone", Path: "/repo/sample.go", Line: 11, Candidate: candidate},
		{Kind: "ccgraph_clone", Path: "sample.go", Line: 12, Candidate: candidate},
	}})
	loaded := m.findingLookup
	if loaded == qualityIndex || m.filePatternCount("sample.go") != 1 || len(m.findingTargets()) != 3 {
		t.Fatal("new results do not replace the index or merge path aliases")
	}
	for _, key := range []string{"]", "j", "k", "p", "]", "p", "["} {
		m = pressSourceKey(t, m, key)
		_ = m.View()
		if m.findingLookup != loaded {
			t.Fatal("navigation rebuilds the report's finding index")
		}
	}
	m = m.resetPatternScan()
	cleared := m.findingLookup
	if cleared == loaded || m.filePatternCount("sample.go") != 0 || len(m.findingTargets()) != 1 || loaded.patterns["sample.go"].count != 1 {
		t.Fatal("refresh retains old findings or mutates the previous index")
	}
	m = m.applyReport(domain.Report{})
	if len(m.findingTargets()) != 0 || m.findingLookup.qualityCount != 0 {
		t.Fatal("a replacement report retains quality markers")
	}
}

func largeFindingModel() Model {
	report := domain.Report{}
	var markers []PatternMarker
	for i := range 585 {
		path := fmt.Sprintf("src/file-%03d.go", i)
		file := domain.File{Path: path}
		for j := range 12 {
			line := 5*j + 1
			location := quality.Location{Path: path, Name: "inspect", Line: line}
			function := domain.Function{Name: fmt.Sprintf("inspect%d", j), Line: line, EndLine: line + 2, Complexity: 1, Source: "func inspect() {\n    inspectNext()\n}"}
			if j < 6 {
				function.Quality = &domain.FunctionQuality{FunctionResult: quality.FunctionResult{Location: location}, Diagnostics: []quality.Diagnostic{{Location: location, RuleID: "mutated_targets", Message: "mutated_targets: 4 exceeds 3"}}}
				markers = append(markers, PatternMarker{Path: path, Line: line + 1, Kind: "guard_clause"})
			}
			file.Functions = append(file.Functions, function)
		}
		report.Files = append(report.Files, file)
	}
	m := Model{width: 160, height: 48, showPatterns: true}.applyReport(report)
	return m.withPatterns(patternsMsg{markers: markers})
}

func BenchmarkLargeFindingNavigation(b *testing.B) {
	m := largeFindingModel()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		m = m.focusAdjacentReview(1)
		_ = m.View()
	}
}

func TestTableViewportKeepsSelectionAndScrollbarAtBothEnds(t *testing.T) {
	m := largeFindingModel()
	for _, index := range []int{0, 292, 584} {
		m.fileIndex = index
		rows := m.fileTable(44, 10)
		plain := strings.Join(rows, "\n")
		if len(rows) != 10 || !strings.Contains(plain, fmt.Sprintf("file-%03d.go", index)) || !strings.Contains(plain, "█") {
			t.Fatalf("viewport hides selected file or scrollbar at %d", index)
		}
	}
}
