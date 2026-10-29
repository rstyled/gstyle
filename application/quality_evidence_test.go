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
		{Site: 42, Line: 2986, Kind: quality.UnknownEffect, Detail: "TARGET_LOCATION"},
		{Site: 42, Line: 2986, Kind: quality.UnknownEffect, Detail: "TARGET_LOCATION"},
	}
	m = m.applyReport(m.report)
	plain := ansi.Strip(strings.Join(m.sourceLines(130, 100, "Source"), "\n"))
	start, end := strings.Index(plain, "2986 │"), strings.Index(plain, "2987 │")
	if start < 0 || end <= start || !strings.Contains(plain[start:end], "╰─■ unknown: TARGET_LOCATION") || strings.Contains(plain[:start], "TARGET_LOCATION") || strings.Count(plain, "TARGET_LOCATION") != 1 {
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

func TestUnknownSummaryPreservesKnownEffectsAndRecordedLines(t *testing.T) {
	m := qualityModel()
	diagnostic := &m.report.Files[0].Functions[0].Quality.Diagnostics[0]
	diagnostic.Effects = []quality.Effect{
		{Line: 12, Kind: quality.UnknownEffect, Detail: "unresolved target one"},
		{Line: 12, Kind: quality.IO, Detail: "confirmed write"},
		{Line: 12, Kind: quality.UnknownEffect, Detail: "unresolved target two"},
		{Line: 13, Kind: quality.UnknownEffect, Detail: "another source line"},
		{Line: 500, Kind: quality.UnknownEffect, Detail: "unavailable target one"},
		{Line: 500, Kind: quality.UnknownEffect, Detail: "unavailable target two"},
	}
	m.report.Files[0].Functions[0].Quality.Effects = diagnostic.Effects
	plain := ansi.Strip(strings.Join(m.sourceLines(130, 100, "Source"), "\n"))
	start, end := strings.Index(plain, "12 │"), strings.Index(plain, "13 │")
	if start < 0 || end <= start || !containsAll(plain[start:end], "io: confirmed write", "2 unresolved effect records") {
		t.Fatalf("summarizing unknowns hides known effects or moves their source location:\n%s", plain)
	}
	if !containsAll(plain[end:], "unknown: another source line") || !containsAll(plain[:start], "Source location unavailable: L500", "2 unresolved effect records") {
		t.Fatalf("summarizing unknowns combines different recorded source lines:\n%s", plain)
	}
	if len(diagnostic.Effects) != 6 || !containsAll(strings.Join(m.qualityBody(), "\n"), "unresolved target one", "unresolved target two") {
		t.Fatal("inline summarization changes the raw evidence")
	}
}

func TestUnknownEstimateDoesNotPretendToBeKnownDensity(t *testing.T) {
	message := qualityBudgetMessage(quality.Diagnostic{
		RuleID: "side_effect_density", Actual: 2166, Limit: 500,
		Weight: 26, Statements: 12, KindWeights: map[string]int64{"unknown": 25, "global_read": 1},
	})
	if message != "Known contribution: 83 · estimate: 2166 · limit: 500" {
		t.Fatalf("a density estimate is displayed as a measured excess: %s", message)
	}
}

func TestUnknownSummaryKeepsDistinctOperationsAndLegacyEvidence(t *testing.T) {
	for _, sites := range [][2]int{{10, 20}, {0, 0}} {
		message := inlineUnknownEvidence(12, []quality.Effect{
			{Site: sites[0], Line: 12, Kind: quality.UnknownEffect, Detail: "same call text"},
			{Site: sites[1], Line: 12, Kind: quality.UnknownEffect, Detail: "same call text"},
		})
		if !strings.Contains(message, "2 unresolved effect records") {
			t.Fatalf("separate operations or legacy records are lost: %s", message)
		}
	}
}

func TestUnknownSummaryCountsEveryRawRecord(t *testing.T) {
	message := inlineUnknownEvidence(12, []quality.Effect{
		{Site: 10, Line: 12, Kind: quality.UnknownEffect, Detail: "first"},
		{Site: 10, Line: 12, Kind: quality.UnknownEffect, Detail: "first"},
		{Site: 10, Line: 12, Kind: quality.UnknownEffect, Detail: "second"},
	})
	if !strings.Contains(message, "3 unresolved effect records") {
		t.Fatalf("the summary loses repeated helper evidence: %s", message)
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
