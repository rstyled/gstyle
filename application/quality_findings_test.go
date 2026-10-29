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
	for _, want := range []string{"╰─■ quality: mutation_per_target", "4 exceeds 3", "Actual: 4", "limit: 3", "╰─■ unknown: callback", "state.count (external)"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("inline quality hides %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "▲") || strings.Contains(plain, "L12 unknown:") || strings.Contains(plain, "L11 state.count") || strings.Count(plain, "quality: mutation_per_target") != 1 {
		t.Fatalf("inline evidence repeats its line or function heading:\n%s", plain)
	}
	if m.showPatterns || m.qualityView || m.filePatternCount("sample.go") != 0 {
		t.Fatal("quality findings need a toggle or change the pattern count")
	}
	m = pressSourceKey(t, m, "]")
	if m.activeFindingLine != 11 || m.qualityView || m.focus != detailsPane {
		t.Fatal("brackets do not reveal inline quality")
	}
}

func TestQualityLocationsDistinguishFunctionChecksFromWriteSites(t *testing.T) {
	m := qualityModel()
	function := m.report.Files[0].Functions[0]
	for _, test := range []struct {
		name       string
		diagnostic quality.Diagnostic
		line       int
		scope      string
	}{
		{"write site", quality.Diagnostic{Location: quality.Location{Path: "sample.go", Line: 10}, RuleID: "mutation_per_target", Mutations: []quality.Mutation{{Line: 13}, {Line: 11}, {Line: 500}}}, 11, "mutation target"},
		{"whole function", quality.Diagnostic{Location: quality.Location{Path: "sample.go", Line: 10}, RuleID: "side_effect_density", Effects: []quality.Effect{{Line: 12}}}, 10, "whole function"},
		{"specific source", quality.Diagnostic{Location: quality.Location{Path: "sample.go", Line: 12}, RuleID: "nilerr"}, 12, "source location"},
		{"missing source", quality.Diagnostic{Location: quality.Location{Path: "sample.go", Line: 500}, RuleID: "nilerr"}, 10, "outside the available function source"},
		{"another file", quality.Diagnostic{Location: quality.Location{Path: "other.go", Line: 500}, RuleID: "mutation_per_target", Mutations: []quality.Mutation{{Line: 11}}}, 500, "recorded source location"},
	} {
		t.Run(test.name, func(t *testing.T) {
			marker := qualityFindingMarker("sample.go", function, test.diagnostic)
			if marker.Line != test.line || !strings.Contains(marker.Scope, test.scope) || marker.Diagnostic.Line != test.diagnostic.Line || marker.Path != test.diagnostic.Path {
				t.Fatalf("finding gets a false source location: %+v", marker)
			}
		})
	}
}

func TestInlineQualityEvidenceStaysAtItsSourceLine(t *testing.T) {
	m := qualityModel()
	q := m.report.Files[0].Functions[0].Quality
	for _, detail := range []string{"first", "first", "second", "third", "fourth", "FINAL_EVIDENCE"} {
		q.Effects = append(q.Effects, quality.Effect{Line: 12, Kind: quality.UnknownEffect, Detail: detail})
	}
	q.Diagnostics[0].Effects = q.Effects
	plain := ansi.Strip(strings.Join(m.sourceLines(150, 100, "Source"), "\n"))
	start, end := strings.Index(plain, "12 │"), strings.Index(plain, "13 │")
	if !containsAll(plain, "estimates, not confirmed", "two", "three") || strings.Count(plain, "unknown: first") != 1 || start < 0 || end <= start {
		t.Fatalf("inline evidence is lost or repeated:\n%s", plain)
	}
	for _, want := range []string{"callback", "first", "second", "third", "fourth", "FINAL_EVIDENCE"} {
		if !strings.Contains(plain[start:end], "unknown: "+want) {
			t.Fatalf("evidence %s does not appear below its source line:\n%s", want, plain)
		}
	}
	if len(q.Effects) != 7 || !strings.Contains(strings.Join(m.qualityBody(), "\n"), "FINAL_EVIDENCE") {
		t.Fatal("compact inline notes discard the full evidence")
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
	for _, want := range []string{"sample.go:10", "sample.go:11", note.ID, "sample.go:12", "sample.go:13", "sample.go:10"} {
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
	for _, want := range []string{"sample.go:11", note.ID, "sample.go:12", "sample.go:10"} {
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
	diagnostic.Message = "side effects need review"
	for _, detail := range []string{strings.Repeat("first evidence ", 35), "FINAL_EVIDENCE"} {
		diagnostic.Effects = append(diagnostic.Effects, quality.Effect{Line: 12, Kind: quality.UnknownEffect, Detail: detail})
	}
	m = pressSourceKey(t, m, "]")
	m = pressSourceKey(t, m, "]")
	for range 200 {
		if strings.Contains(ansi.Strip(m.View().Content), "FINAL_EVIDENCE") {
			break
		}
		m = pressSourceKey(t, m, "j")
	}
	if m.sourceLine() != 12 || m.findingDetailOffset == 0 || !strings.Contains(ansi.Strip(m.View().Content), "FINAL_EVIDENCE") {
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
