package application

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/domain"
)

type loadingAnalyzer struct {
	report domain.Report
}

func (a loadingAnalyzer) Analyze([]string) (domain.Report, error) {
	return a.report, nil
}

func TestStartupLoadsPatternsAndInlineQualityAutomatically(t *testing.T) {
	m := NewModel(loadingAnalyzer{qualityModel().report}, []string{"../domain/report.go"})
	m.width, m.height = 160, 40
	if !m.showPatterns || !strings.Contains(ansi.Strip(m.header(160)), "loading code + quality") {
		t.Fatal("startup hides patterns or does not show its loading phase")
	}
	loaded, command := m.Update(m.Init()())
	m = loaded.(Model)
	if command == nil || !m.patternsLoading || m.refreshing || !strings.Contains(ansi.Strip(m.header(160)), "loading patterns") {
		t.Fatal("the report does not start pattern loading automatically")
	}
	if !strings.Contains(ansi.Strip(strings.Join(m.sourceLines(100, 40, "Source"), "\n")), "quality: mutation_per_target") {
		t.Fatal("quality findings wait for pattern loading or a toggle")
	}
	result := command().(patternsMsg)
	if result.err != nil || result.findings == nil {
		t.Fatalf("the worker does not prepare the index: %v", result.err)
	}
	loaded, followup := m.Update(result)
	m = loaded.(Model)
	if followup != nil || m.patternsLoading || m.patternsError != nil || m.analysisLoadingLabel() != "" || m.findingLookup != result.findings {
		t.Fatalf("automatic pattern loading does not finish with its prepared index: %v", m.patternsError)
	}
	result.generation--
	result.findings = &findingIndex{}
	if m.withPatterns(result).findingLookup != m.findingLookup {
		t.Fatal("a stale worker index replaces the current report")
	}
}

func TestPatternVisibilityDoesNotStartASecondScanDuringInitialLoad(t *testing.T) {
	m := NewModel(loadingAnalyzer{qualityModel().report}, nil)
	generation := m.patternGeneration
	for range 2 {
		next, command, handled := m.updateGlobalKey("p")
		m = next
		if !handled || command != nil || m.patternsLoading || m.patternGeneration != generation {
			t.Fatal("toggling visibility starts a concurrent analysis before the report is ready")
		}
	}
}

func TestRefreshRejectsOldPatternResultsAndStartsANewScan(t *testing.T) {
	m := NewModel(loadingAnalyzer{qualityModel().report}, nil)
	loaded, _ := m.Update(m.Init()())
	m = loaded.(Model)
	oldGeneration := m.patternGeneration
	refreshed, command, _ := m.updateGlobalKey("r")
	revision := refreshed.revision
	loaded, _ = refreshed.Update(patternsMsg{generation: oldGeneration, markers: []PatternMarker{{Kind: "stale"}}})
	m = loaded.(Model)
	if len(m.patternMarkers) != 0 || m.revision != revision || !m.refreshing {
		t.Fatal("old pattern results overwrite an active refresh")
	}
	loaded, next := m.Update(command())
	m = loaded.(Model)
	if next == nil || !m.patternsLoading || m.patternGeneration == oldGeneration {
		t.Fatal("refresh does not start a new pattern scan")
	}
}
