package application

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/domain"
)

func mixedReviewModel() Model {
	m := annotationCycleModel()
	m.showPatterns = true
	m.report.Files[1].Functions = append(m.report.Files[1].Functions, domain.Function{
		Name: "second", Line: 30, EndLine: 32, Source: "func second() {\n    return\n}",
	})
	m.patternMarkers = []PatternMarker{
		{Kind: "guard_clause", Path: "a.go", Line: 1},
		{Kind: "ccgraph_clone", Path: "a.go", Line: 3},
		{Kind: "ccgraph_clone", Path: "/repo/b.go", Line: 31},
	}
	return m
}

func reviewSelection(m Model) string {
	if m.activeAnnotationID != "" {
		return m.activeAnnotationID
	}
	return fmt.Sprintf("%s:%d", m.selectedFilePath(), m.sourceLine())
}

func TestBracketKeysCycleNotesAndPatternsAcrossAllPanes(t *testing.T) {
	forward := []string{"a.go:1", "n1", "a.go:3", "n2", "n3", "n4", "b.go:31", "a.go:1"}
	backward := []string{"b.go:31", "n4", "n3", "n2", "a.go:3", "n1", "a.go:1", "b.go:31"}
	for _, focus := range []pane{filesPane, functionsPane, detailsPane} {
		m := mixedReviewModel()
		m.focus, m.qualityView = focus, true
		for _, sequence := range []struct {
			key   string
			stops []string
		}{{"]", forward}, {"[", backward}} {
			for _, want := range sequence.stops {
				revision := m.revision
				m = pressSourceKey(t, m, sequence.key)
				if got := reviewSelection(m); got != want {
					t.Fatalf("pane=%d key=%s: got %s, want %s", focus, sequence.key, got, want)
				}
				if m.focus != detailsPane || m.qualityView || m.revision != revision+1 || m.lineSelection != nil {
					t.Fatal("review navigation does not reveal source with one revision change")
				}
			}
		}
		if !strings.Contains(ansi.Strip(m.footer(200)), "findings (7/7; 3 unmatched)") {
			t.Fatalf("footer does not show the combined cycle: %s", ansi.Strip(m.footer(200)))
		}
	}
}

func TestPatternStopsDeduplicateLocationsWithoutSkippingSameLineNotes(t *testing.T) {
	m := annotationCycleModel()
	m.annotations = m.annotations[:1]
	m.showPatterns = true
	m.patternMarkers = []PatternMarker{
		{Kind: "ccgraph_clone", Path: "a.go", Line: 2},
		{Kind: "guard_clause", Path: "a.go", Line: 2},
		{Kind: "ccgraph_clone", Path: "a.go", Line: 2},
		{Kind: "guard_clause", Path: "a.go", Line: 500},
		{Kind: "guard_clause", Path: "gone.go", Line: 1},
	}
	for _, want := range []string{"a.go:2", "n1", "a.go:2", "n1"} {
		m = pressSourceKey(t, m, "]")
		if got := reviewSelection(m); got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
	if len(m.annotations) != 1 || len(m.patternMarkers) != 5 {
		t.Fatal("navigation changes saved notes or findings")
	}
}

func TestHiddenPatternsAreSkippedAndManualMovementClearsPatternStop(t *testing.T) {
	m := mixedReviewModel()
	m = pressSourceKey(t, m, "]")
	m = pressSourceKey(t, m, "j")
	if m.activeFindingLine != 0 {
		t.Fatal("manual movement retains a stale pattern stop")
	}
	m = pressSourceKey(t, m, "]")
	if m.activeAnnotationID != "n1" {
		t.Fatal("manual movement does not resume from the cursor")
	}
	m = pressSourceKey(t, m, "p")
	m = pressSourceKey(t, m, "]")
	if m.activeAnnotationID != "n2" || m.showPatterns {
		t.Fatal("hidden patterns interrupt the saved-note cycle")
	}
	if strings.Contains(ansi.Strip(m.footer(200)), "findings (") {
		t.Fatal("footer counts hidden pattern stops")
	}
}

func TestPatternNavigationUsesInnermostAvailableSourceAndKeepsCursorVisible(t *testing.T) {
	m := sourceWorkspaceModel()
	m.report.Files[0].Functions = append(m.report.Files[0].Functions, domain.Function{
		Name: "nested", Line: 15, EndLine: 17, Source: "nested {\n    call()\n}",
	})
	m.showPatterns = true
	m.patternMarkers = []PatternMarker{{Kind: "guard_clause", Path: "inspect.go", Line: 16}}
	m = pressSourceKey(t, m, "]")
	if m.functionIndex != 1 || m.sourceLine() != 16 || m.activeFindingLine != 16 {
		t.Fatal("pattern navigation selects the wrong source scope")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "›   16 │") {
		t.Fatal("pattern navigation hides the cursor")
	}
	m = updateSourceModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.activeFindingLine != 0 {
		t.Fatal("escape retains an active pattern stop")
	}
}

func TestBracketKeysWithNoReviewTargetsDoNothing(t *testing.T) {
	m := Model{showPatterns: true, patternMarkers: []PatternMarker{{Path: "missing.go", Line: 1}}}
	for _, key := range []string{"[", "]"} {
		m = pressSourceKey(t, m, key)
		if m.revision != 0 || m.focus != filesPane {
			t.Fatal("empty review navigation changes the view")
		}
	}
}
