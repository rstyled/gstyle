package application

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/domain/patterns"
)

func TestPatternNotesShowEvidenceReviewAndRelatedLocations(t *testing.T) {
	report := &patterns.PatternsReport{Candidates: []patterns.Candidate{{
		Kind: patterns.CCGraphClone, Observation: "Two functions have similar dependency graphs.",
		Inference:        "They may share behavior; similarity does not prove equivalence.",
		PossibleRefactor: "Compare their effects before extracting a shared helper.",
		CounterEvidence:  []string{"One function serves a different platform."},
		Sites: []patterns.Site{
			{Path: "a.go", Line: 10, Name: "first"},
			{Path: "b.go", Line: 30, Name: "second"},
		},
		FixSpec: &patterns.FixSpec{Kind: patterns.Parameterize},
	}}}
	markers := patternMarkersFromReport(report)
	if markers[0].Candidate != markers[1].Candidate || markers[0].Candidate.FixSpec != nil {
		t.Fatal("display markers lose shared finding identity or retain source-edit context")
	}
	rows := sourcePatternRows(newFindingFile(markers).starts[10], 55)
	plain := ansi.Strip(strings.Join(rows, "\n"))
	for _, want := range []string{"pattern: ccgraph_clone", "Observed:", "Inference:", "Review:", "Related: second (b.go:30)", "Caution:"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("pattern note hides %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "Related: first") {
		t.Fatal("pattern note repeats its own location as related evidence")
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > 55 {
			t.Fatalf("pattern note exceeds the pane width: %q", ansi.Strip(row))
		}
	}
}

func TestSameKindFindingsAtOneLineKeepTheirOwnDetails(t *testing.T) {
	report := &patterns.PatternsReport{Candidates: []patterns.Candidate{
		{Kind: patterns.CCGraphClone, Observation: "first clone group", Sites: []patterns.Site{{Path: "a.go", Line: 10}, {Path: "a.go", Line: 10}}},
		{Kind: patterns.CCGraphClone, Observation: "second clone group", Sites: []patterns.Site{{Path: "a.go", Line: 10}}},
	}}
	markers := newFindingFile(patternMarkersFromReport(report)).starts[10]
	plain := ansi.Strip(strings.Join(sourcePatternRows(markers, 80), "\n"))
	if strings.Count(plain, "pattern: ccgraph_clone") != 2 || !containsAll(plain, "first clone group", "second clone group") {
		t.Fatalf("same-kind findings hide each other's evidence:\n%s", plain)
	}
}

func TestCloneNotesIdentifyWholeFunctionAndRelatedRange(t *testing.T) {
	report := &patterns.PatternsReport{Candidates: []patterns.Candidate{{
		Kind: patterns.CCGraphClone, Observation: "2 function bodies have similar dependency graphs",
		Sites: []patterns.Site{
			{ID: "reader", Name: "Reader.decode", Path: "reader.go", Line: 929, EndLine: 954},
			{ID: "our-reader", Name: "OurReader.decode", Path: "reader.go", Line: 1900, EndLine: 1925},
		},
	}}}
	markers := patternMarkersFromReport(report)
	for _, test := range []struct {
		line        int
		scope, peer string
	}{
		{929, "lines 929-954", "OurReader.decode (reader.go:1900-1925)"},
		{1900, "lines 1900-1925", "Reader.decode (reader.go:929-954)"},
	} {
		at := newFindingFile(markers).starts[test.line]
		plain := ansi.Strip(strings.Join(sourcePatternRows(at, 150), "\n"))
		if !containsAll(plain, "Scope: whole function ("+test.scope+")", "Related: "+test.peer) {
			t.Fatalf("clone note hides its function scope or counterpart:\n%s", plain)
		}
		if !at[0].covers("reader.go", test.line+10) {
			t.Fatal("clone highlight loses the function body range")
		}
	}
	if markers[1].covers("reader.go", 1926) {
		t.Fatal("clone highlight extends past the closing brace")
	}
	guard := FindingMarker{Kind: string(patterns.GuardClause), Line: 1903, EndLine: 1906}
	if len(guard.patternScopeMessages()) != 0 {
		t.Fatal("statement finding is labeled as a whole-function comparison")
	}
}

func TestPatternDetailRowsMatchSourceScrollMeasurements(t *testing.T) {
	m := sourceWorkspaceModel()
	report := &patterns.PatternsReport{Candidates: []patterns.Candidate{{
		Kind: patterns.GuardClause, Observation: strings.Repeat("nested branch ", 12),
		PossibleRefactor: "Use an early return if it preserves the branch behavior.",
		Sites:            []patterns.Site{{Path: "inspect.go", Line: 12, EndLine: 16}},
	}}}
	m.showPatterns, m.patternMarkers = true, patternMarkersFromReport(report)
	function, _ := m.selectedFunction()
	for _, width := range []int{60, 120, 200} {
		m.width = width
		for offset := range m.selectedSourceLineCount() {
			m.sourceOffset = offset
			rows := m.sourceDisplayWindow(m.selectedSourceCodeLines(function), function, paneContentWidth(m.sourcePaneWidth()), 10000)
			if got := m.sourceDisplayRowCount(offset, m.selectedSourceLineCount()-1); got != len(rows) {
				t.Fatalf("width=%d offset=%d: measured %d rows, rendered %d", width, offset, got, len(rows))
			}
		}
	}
}

func TestTallPatternNotesRemainReadableAndCanBeLeft(t *testing.T) {
	m := sourceWorkspaceModel()
	m.width, m.height = 90, 18
	m.showPatterns = true
	m.patternMarkers = patternMarkersFromReport(&patterns.PatternsReport{Candidates: []patterns.Candidate{{
		Kind: patterns.CCGraphClone, Observation: strings.Repeat("compare the source ", 35),
		CounterEvidence: []string{"FINAL_CAVEAT"}, Sites: []patterns.Site{{Path: "inspect.go", Line: 12}},
	}}})
	m = pressSourceKey(t, m, "]")
	if !containsAll(ansi.Strip(m.View().Content), "pattern:", "ccgraph_clone") {
		t.Fatal("pattern navigation hides the note header")
	}
	for range 200 {
		if strings.Contains(ansi.Strip(m.View().Content), "FINAL_CAVEAT") {
			break
		}
		revision := m.revision
		m = pressSourceKey(t, m, "j")
		if m.revision != revision+1 || m.sourceLine() != 12 {
			t.Fatal("scrolling a tall pattern note skips its remaining rows")
		}
	}
	if m.findingDetailOffset == 0 || !strings.Contains(ansi.Strip(m.View().Content), "FINAL_CAVEAT") {
		t.Fatal("the end of a tall pattern note is unreachable")
	}
	m = pressSourceKey(t, m, "k")
	if m.findingDetailOffset == 0 || m.sourceLine() != 12 {
		t.Fatal("scrolling back leaves the pattern note")
	}
	m = pressSourceKey(t, m, "]")
	if m.findingDetailOffset != 0 || m.activeFindingLine != 12 {
		t.Fatal("bracket navigation does not reset pattern scrolling")
	}
	for range 200 {
		m = pressSourceKey(t, m, "j")
		if m.sourceLine() == 13 {
			break
		}
	}
	if m.sourceLine() != 13 || m.activeFindingLine != 0 || m.findingDetailOffset != 0 {
		t.Fatal("cannot leave a fully scrolled pattern note")
	}
}
