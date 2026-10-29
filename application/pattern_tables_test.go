package application

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/patterns"
)

func patternTableModel() Model {
	m := sourceWorkspaceModel()
	m.report.Files[0].Functions = append(m.report.Files[0].Functions, domain.Function{
		Name: "second", Line: 30, EndLine: 32, Source: "func second() {\n    return\n}",
	})
	m.showPatterns = true
	m.patternMarkers = patternMarkersFromReport(&patterns.PatternsReport{Candidates: []patterns.Candidate{
		{Kind: patterns.CCGraphClone, Sites: []patterns.Site{{Path: "/repo/inspect.go", Line: 11}, {Path: "/repo/inspect.go", Line: 15}, {Path: "other.go", Line: 31}}},
		{Kind: patterns.CCGraphClone, Sites: []patterns.Site{{Path: "inspect.go", Line: 11}, {Path: "other.go", Line: 31}}},
		{Kind: patterns.GuardClause, Sites: []patterns.Site{{Path: "inspect.go", Line: 16}}},
		{Kind: patterns.TypedNil, Sites: []patterns.Site{{Path: "inspect.go", Line: 31}}},
	}})
	return m
}

func TestPATColumnsCountDistinctFindingsPerFileAndFunction(t *testing.T) {
	m := patternTableModel()
	file, _ := m.selectedFile()
	if m.filePatternCount(file.Path) != 4 || m.filePatternCount("other.go") != 2 {
		t.Fatal("file counts include sites instead of distinct findings")
	}
	if m.functionPatternCount(file.Path, file.Functions[0]) != 3 || m.functionPatternCount(file.Path, file.Functions[1]) != 1 {
		t.Fatal("function counts include unrelated findings or duplicate sites")
	}
	if !strings.Contains(ansi.Strip(m.sourceTitle("Source", file.Functions[0])), "▲ 3 patterns") {
		t.Fatal("source title counts patterns outside the selected function")
	}
	for _, width := range []int{24, 29, 30, 41, 42, 64} {
		rows := []string{
			fileTableHeader(width), functionTableHeader(width),
			m.fileRow(file, width, false), m.functionRow(file.Functions[0], width, false),
		}
		for _, row := range rows {
			if ansi.StringWidth(row) > width {
				t.Fatalf("width=%d: table row overflows: %q", width, ansi.Strip(row))
			}
		}
		if !strings.HasSuffix(strings.TrimSpace(ansi.Strip(rows[2])), "4") || !strings.HasSuffix(strings.TrimSpace(ansi.Strip(rows[3])), "3") {
			t.Fatal("PAT cells do not show their row's finding count")
		}
	}
}

func TestPATColumnsDistinguishHiddenLoadingAndZero(t *testing.T) {
	for _, tc := range []struct {
		show, loading bool
		want          string
	}{{false, false, "-"}, {true, true, "…"}, {true, false, "0"}} {
		m := Model{showPatterns: tc.show, patternsLoading: tc.loading}
		if got := strings.TrimSpace(ansi.Strip(m.patternCountCell(0))); got != tc.want {
			t.Fatalf("show=%t loading=%t: got %q, want %q", tc.show, tc.loading, got, tc.want)
		}
	}
}

func TestPatternFailureDoesNotLookLikeZeroFindings(t *testing.T) {
	m := sourceWorkspaceModel()
	m.showPatterns, m.patternsLoading = true, true
	revision := m.revision
	m = updateSourceModel(t, m, patternsMsg{err: errors.New("pattern input failed")})
	if m.patternsLoading || m.revision != revision+1 || strings.TrimSpace(ansi.Strip(m.patternCountCell(0))) != "!" {
		t.Fatal("failed detection displays a clean count or does not notify the view")
	}
	function, _ := m.selectedFunction()
	if !strings.Contains(ansi.Strip(m.sourceTitle("Source", function)), "patterns failed: pattern input failed") {
		t.Fatal("failed detection hides its cause")
	}
}
