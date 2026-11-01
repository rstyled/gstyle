package application

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestQualityTogglePreservesSourceAndOtherHighlights(t *testing.T) {
	for _, focus := range []pane{filesPane, functionsPane, detailsPane} {
		m := qualityModel()
		m.focus, m.sourceCursor = focus, 2
		m.lineSelection = &LineSelection{StartLine: 11, EndLine: 12}
		m.showPatterns = true
		m.patternMarkers = []PatternMarker{{Kind: "guard_clause", Path: "sample.go", Line: 13}}
		m = m.ensureSourceRows()
		lookup, rows, revision := m.findingLookup, m.sourceRows, m.revision

		m = pressSourceKey(t, m, "e")
		if m.showQuality || m.qualityView || m.focus != focus || m.sourceCursor != 2 || !m.showPatterns {
			t.Fatal("quality toggle changes source navigation or patterns")
		}
		if m.lineSelection == nil || m.lineSelection.StartLine != 11 || m.revision <= revision {
			t.Fatal("quality toggle loses selection or revision")
		}
		if m.findingLookup != lookup || m.sourceRows == rows {
			t.Fatal("quality toggle rebuilds analysis or retains stale display rows")
		}
		view := strings.Join(m.sourceLines(100, 0, "Source"), "\n")
		if strings.Contains(ansi.Strip(view), "quality: mutation_per_target") || !strings.Contains(ansi.Strip(view), "pattern: guard_clause") {
			t.Fatal("quality toggle hides patterns or leaves quality labels")
		}

		m = pressSourceKey(t, m, "e")
		m.sourceOffset = 0
		view = strings.Join(m.sourceLines(100, 0, "Source"), "\n")
		if !m.showQuality || !strings.Contains(view, qualityBackground) || !strings.Contains(ansi.Strip(view), "Q quality: mutation_per_target") {
			t.Fatal("quality toggle does not restore inline blue findings")
		}
		if !strings.Contains(ansi.Strip(view), "func Change()") || strings.Contains(ansi.Strip(view), "Known effects only") {
			t.Fatal("quality toggle replaces source with the quality report")
		}
	}
}

func TestFindingTogglesControlReviewNavigationIndependently(t *testing.T) {
	m := qualityModel()
	m.patternMarkers = []PatternMarker{{Kind: "guard_clause", Path: "sample.go", Line: 13}}
	note, _ := m.newAnnotation(12, 12, "saved note")
	m.annotations = []Annotation{note}
	for _, state := range []struct {
		patterns, quality bool
		findings, targets int
	}{
		{true, true, 2, 3},
		{true, false, 1, 2},
		{false, true, 1, 2},
		{false, false, 0, 1},
	} {
		m.showPatterns, m.showQuality = state.patterns, state.quality
		if len(m.findingTargets()) != state.findings || len(m.reviewTargets()) != state.targets {
			t.Fatalf("wrong review targets with patterns=%v quality=%v", state.patterns, state.quality)
		}
		if m.hasFindingAtLine(10) != state.quality || m.hasFindingAtLine(13) != state.patterns {
			t.Fatal("hidden findings retain a source background")
		}
		if (len(m.findingsStartingAt(10)) > 0) != state.quality || (len(m.findingsStartingAt(13)) > 0) != state.patterns {
			t.Fatal("hidden findings retain inline rows")
		}
	}
	m = pressSourceKey(t, m, "]")
	if m.activeAnnotationID != note.ID {
		t.Fatal("hiding all findings blocks saved-note navigation")
	}
}

func TestQualityToggleLeavesHiddenFindingAndKeepsCursorVisible(t *testing.T) {
	m := qualityModel()
	m.width, m.height = 90, 18
	m.report.Files[0].Functions[0].Quality.Diagnostics[0].Message = strings.Repeat("long finding ", 50)
	m = pressSourceKey(t, m, "]")
	m = pressSourceKey(t, m, "j")
	if m.findingDetailOffset == 0 {
		t.Fatal("test does not scroll the quality finding")
	}
	m = pressSourceKey(t, m, "e")
	if m.activeFindingLine != 0 || m.findingDetailOffset != 0 {
		t.Fatal("hidden quality finding captures source scrolling")
	}
	m = pressSourceKey(t, m, "j")
	if m.sourceLine() != 11 {
		t.Fatal("source movement does not resume after hiding quality")
	}
	m = pressSourceKey(t, m, "e")
	if m.sourceLine() != 11 || !strings.Contains(ansi.Strip(m.View().Content), "›   11 │") {
		t.Fatal("restoring quality hides or moves the source cursor")
	}
}
