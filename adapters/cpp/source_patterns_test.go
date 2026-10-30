package cpp

import (
	"context"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestRealCPPSourcePatterns(t *testing.T) {
	extraction, err := Load(context.Background(), "testdata/patterns.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	report := extraction.Patterns()
	want := map[patterns.CandidateKind]map[string]int{
		patterns.GuardClause:        {"guard_candidate": 5},
		patterns.ConstantBoolReturn: {"same_explicit": 25, "same_tail": 29, "condition_binding": 50},
		patterns.CPPEmptyCatch:      {"empty_catch": 54, "typed_empty_catch": 58},
	}
	graphCounts := map[patterns.CandidateKind]int{}
	for _, candidate := range report.Candidates {
		if candidate.FixSpec != nil || candidate.Analysis == nil || candidate.Analysis.Status != "complete" || candidate.Analysis.Scope != patterns.ImportedGraphScope {
			t.Fatalf("imported pattern lacks a complete scoped query or claims a fix: %+v", candidate)
		}
		sites, source := want[candidate.Kind]
		if !source {
			if candidate.ScoreMilli > 400 {
				t.Fatalf("graph measurement outranks source findings: %+v", candidate)
			}
			continue
		}
		site := candidate.Sites[0]
		name, _, _ := strings.Cut(site.Name, ":")
		line, exists := sites[name]
		if !exists || site.Line != line {
			t.Fatalf("unexpected %s at %s:%d", candidate.Kind, name, site.Line)
		}
		delete(sites, name)
		if candidate.Kind == patterns.CPPEmptyCatch && candidate.Kind.Group() != patterns.CPPPatterns {
			t.Fatal("C++ handler rule is classified as general")
		}
	}
	for kind, sites := range want {
		if len(sites) != 0 {
			t.Errorf("missing %s findings: %v", kind, sites)
		}
	}
	for _, measurement := range report.Measurements {
		graphCounts[measurement.Kind]++
	}
	for _, kind := range []patterns.CandidateKind{patterns.ThinSliceKind, patterns.ChopKind, patterns.DependenceCluster} {
		if graphCounts[kind] == 0 {
			t.Errorf("real C++ dependencies do not reach %s", kind)
		}
	}
	if len(report.AnalysisCoverage) != 27 {
		t.Fatalf("coverage omits enabled rules: %+v", report.AnalysisCoverage)
	}
}

func TestCPPConditionScope(t *testing.T) {
	for source, want := range map[string]bool{
		"if (ready) {}":                               true,
		"if (check(value)) {}":                        true,
		"if (int value = check(); value) {}":          true,
		"if (text == \"(\") {}":                       true,
		"if ([] { return true; }()) {}":               true,
		"if /* () */ (int value = check(); value) {}": true,
		"if /* note */ constexpr (Enabled) {}":        false,
		"if consteval {}":                             false,
		"if !consteval {}":                            false,
		"if (ready) { /* constexpr */ }":              true,
	} {
		if got := runtimeIfCode(source); got != want {
			t.Errorf("%s: got %t, want %t", source, got, want)
		}
	}
}
