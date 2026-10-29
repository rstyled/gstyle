package application

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/domain/quality"
)

func TestQualityGuardrailsAreInlineWithoutToggles(t *testing.T) {
	m := qualityModel()
	diagnostic := &m.report.Files[0].Functions[0].Quality.Diagnostics[0]
	diagnostic.Effects = []quality.Effect{{Line: 12, Kind: quality.UnknownEffect, Detail: "callback"}}
	diagnostic.Mutations = m.report.Files[0].Functions[0].Quality.MutationEvents
	plain := ansi.Strip(strings.Join(m.sourceLines(100, 40, "Source"), "\n"))
	for _, want := range []string{"quality: mutation_per_target", "4 exceeds 3", "Actual: 4", "limit: 3", "L12 unknown: callback", "L11 state.count (external)"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("inline quality hides %q:\n%s", want, plain)
		}
	}
	if m.showPatterns || m.qualityView || m.filePatternCount("sample.go") != 0 {
		t.Fatal("quality findings need a toggle or change the pattern count")
	}
	m = pressSourceKey(t, m, "]")
	if m.activeFindingLine != 10 || m.qualityView || m.focus != detailsPane {
		t.Fatal("brackets do not reveal inline quality")
	}
}

func TestBracketsVisitQualityPatternsAndNotesTogether(t *testing.T) {
	m := qualityModel()
	m.showPatterns = true
	m.patternMarkers = []PatternMarker{{Kind: "guard_clause", Path: "sample.go", Line: 13}}
	q := m.report.Files[0].Functions[0].Quality
	q.Diagnostics = append(q.Diagnostics, quality.Diagnostic{
		Location: quality.Location{Path: "sample.go", Line: 12}, RuleID: "mutated_targets", Message: "too many mutated targets",
	})
	note, _ := m.newAnnotation(11, 11, "saved note")
	m.annotations = []Annotation{note}
	for _, want := range []string{"sample.go:10", note.ID, "sample.go:12", "sample.go:13", "sample.go:10"} {
		m = pressSourceKey(t, m, "]")
		if got := reviewSelection(m); got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
	m = pressSourceKey(t, m, "d")
	if len(m.annotations) != 1 || len(q.Diagnostics) != 2 {
		t.Fatal("a quality finding acts as a saved annotation")
	}
	m = pressSourceKey(t, m, "p")
	for _, want := range []string{note.ID, "sample.go:12", "sample.go:10"} {
		m = pressSourceKey(t, m, "]")
		if got := reviewSelection(m); got != want {
			t.Fatalf("with patterns hidden: got %s, want %s", got, want)
		}
	}
}

func TestSameRuleQualityFindingsKeepSeparateEvidence(t *testing.T) {
	m := qualityModel()
	q := m.report.Files[0].Functions[0].Quality
	second := q.Diagnostics[0]
	second.Message = "a second target has its own mutation evidence"
	q.Diagnostics = append(q.Diagnostics, second)
	markers := patternMarkersStartingAt(m.findingMarkers(), "sample.go", 10)
	if len(markers) != 2 || len(m.findingTargets()) != 1 {
		t.Fatal("quality findings are lost or create repeated same-line stops")
	}
	plain := ansi.Strip(strings.Join(sourcePatternRows(markers, 100), "\n"))
	if !containsAll(plain, "4 exceeds 3", second.Message) {
		t.Fatal("same-rule findings hide each other's evidence")
	}
}

func TestInlineQualityScrollExtentMatchesRows(t *testing.T) {
	m := qualityModel()
	m.report.Files[0].Functions[0].Quality.Diagnostics[0].Message = strings.Repeat("guardrail evidence ", 20)
	function, _ := m.selectedFunction()
	for _, width := range []int{40, 90, 160} {
		m.width = width
		for offset := range m.selectedSourceLineCount() {
			m.sourceOffset = offset
			rows := m.sourceDisplayWindow(m.selectedSourceCodeLines(function), function, paneContentWidth(m.sourcePaneWidth()), 10000)
			if got := m.sourceDisplayRowCount(offset, m.selectedSourceLineCount()-1); got != len(rows) {
				t.Fatalf("width=%d offset=%d: measured %d, rendered %d", width, offset, got, len(rows))
			}
		}
	}
}

func TestTallQualityFindingIsScrollableWithoutEnablingPatterns(t *testing.T) {
	m := qualityModel()
	m.width, m.height = 90, 18
	diagnostic := &m.report.Files[0].Functions[0].Quality.Diagnostics[0]
	diagnostic.Message = strings.Repeat("guardrail evidence ", 35)
	diagnostic.Effects = []quality.Effect{{Line: 12, Kind: quality.UnknownEffect, Detail: "FINAL_EVIDENCE"}}
	m = pressSourceKey(t, m, "]")
	for range 200 {
		if strings.Contains(ansi.Strip(m.View().Content), "FINAL_EVIDENCE") {
			break
		}
		m = pressSourceKey(t, m, "j")
	}
	if m.sourceLine() != 10 || m.findingDetailOffset == 0 || !strings.Contains(ansi.Strip(m.View().Content), "FINAL_EVIDENCE") {
		t.Fatal("quality evidence is unreachable without a toggle")
	}
}

func TestVisualSelectionLeavesFindingScrolling(t *testing.T) {
	m := qualityModel()
	m.width, m.height = 90, 18
	m.report.Files[0].Functions[0].Quality.Diagnostics[0].Message = strings.Repeat("guardrail evidence ", 35)
	m = pressSourceKey(t, m, "]")
	m = pressSourceKey(t, m, "j")
	if m.findingDetailOffset == 0 {
		t.Fatal("test does not scroll the finding")
	}
	m = pressSourceKey(t, m, "v")
	m = pressSourceKey(t, m, "j")
	if !m.visualSelectionActive || m.activeFindingLine != 0 || m.findingDetailOffset != 0 || m.sourceLine() != 11 {
		t.Fatal("finding scrolling captures visual-selection movement")
	}
	if m.lineSelection.StartLine != 10 || m.lineSelection.EndLine != 11 {
		t.Fatal("visual selection does not use the selected source lines")
	}
}
