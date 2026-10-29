package application

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/quality"
)

func TestEvidenceAt2986AppearsBelow2986AndIsNavigable(t *testing.T) {
	m := qualityModel()
	function := &m.report.Files[0].Functions[0]
	function.Line, function.EndLine = 2979, 2988
	function.Source = "func inspect() {\n one()\n two()\n three()\n four()\n first()\n six()\n target()\n eight()\n}"
	diagnostic := &function.Quality.Diagnostics[0]
	diagnostic.Location.Line, diagnostic.RuleID, diagnostic.Message = 2979, "side_effect_density", "function density exceeds its limit"
	diagnostic.Effects = []quality.Effect{
		{Line: 2984, Kind: quality.IO, Detail: "FIRST_LOCATION"},
		{Line: 2986, Kind: quality.UnknownEffect, Detail: "TARGET_LOCATION"},
		{Line: 2986, Kind: quality.UnknownEffect, Detail: "TARGET_LOCATION"},
	}
	m = m.applyReport(m.report)
	plain := ansi.Strip(strings.Join(m.sourceLines(130, 100, "Source"), "\n"))
	start, end := strings.Index(plain, "2986 │"), strings.Index(plain, "2987 │")
	if start < 0 || end <= start || !strings.Contains(plain[start:end], "TARGET_LOCATION") || strings.Contains(plain[:start], "TARGET_LOCATION") || strings.Count(plain, "TARGET_LOCATION") != 1 {
		t.Fatalf("evidence is displayed at the function header instead of L2986:\n%s", plain)
	}
	for _, want := range []int{2979, 2984, 2986, 2979} {
		m = pressSourceKey(t, m, "]")
		if m.sourceLine() != want {
			t.Fatalf("quality navigation visits %d, want %d", m.sourceLine(), want)
		}
	}
	m = pressSourceKey(t, m, "[")
	if m.sourceLine() != 2986 || len(function.Quality.Diagnostics) != 1 {
		t.Fatal("evidence changes the finding count or cannot be visited backwards")
	}
}

func TestUnavailableEvidenceIsExplicitAndPreservesItsRecordedLine(t *testing.T) {
	m := qualityModel()
	diagnostic := &m.report.Files[0].Functions[0].Quality.Diagnostics[0]
	diagnostic.Effects = []quality.Effect{{Line: 500, Kind: quality.UnknownEffect, Detail: "missing source"}}
	plain := ansi.Strip(strings.Join(m.sourceLines(120, 40, "Source"), "\n"))
	if !containsAll(plain, "Source location unavailable:", "L500 unknown: missing source") || len(m.findingTargets()) != 1 {
		t.Fatal("unavailable evidence is hidden or gets an invented source stop")
	}
}

func TestSourceRowsCacheFollowsResizeNotesAndFindings(t *testing.T) {
	m := qualityModel().ensureSourceRows()
	first := m.sourceRows
	m = pressSourceKey(t, m, "]")
	m = pressSourceKey(t, m, "j")
	if m.sourceRows != first {
		t.Fatal("same-function navigation rebuilds source rows")
	}
	m.width = 200
	m = m.ensureSourceRows()
	resized := m.sourceRows
	if resized == first {
		t.Fatal("resize retains rows wrapped for the old pane")
	}
	note, _ := m.newAnnotation(11, 11, "a saved note")
	var err error
	m, err = m.saveAnnotation(note)
	if err != nil {
		t.Fatal(err)
	}
	m = m.ensureSourceRows()
	if m.sourceRows == resized || m.sourceRows.count(1, 1) <= resized.count(1, 1) {
		t.Fatal("saved notes do not invalidate source measurements")
	}
	withNote := m.sourceRows
	m = m.withPatterns(patternsMsg{markers: []PatternMarker{{Kind: "guard_clause", Path: "sample.go", Line: 11}}}).ensureSourceRows()
	if m.sourceRows == withNote {
		t.Fatal("new findings do not invalidate source measurements")
	}
	m = m.applyReport(domain.Report{})
	if m.ensureSourceRows().sourceRows == withNote {
		t.Fatal("report replacement retains the previous source rows")
	}
}
