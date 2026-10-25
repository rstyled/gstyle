package application

import (
	"math"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/quality"
)

func searchFixture() domain.Report {
	return domain.Report{Root: "/repo", Files: []domain.File{
		{Path: "calm.go", Total: 1, Functions: []domain.Function{{Name: "Calm", Complexity: 1, Line: 1}}},
		{Path: "hot.go", Total: 8, Functions: []domain.Function{
			{Name: "Simple", Package: "example", Complexity: 2, Line: 20, Source: "return store.Save()"},
			{Name: "Branchy", Package: "example", Complexity: 6, CognitiveComplexity: 8, Line: 4, Quality: &domain.FunctionQuality{
				Diagnostics: []quality.Diagnostic{
					{RuleID: "side_effect_density", Message: "IO boundary", Actual: 1000, Limit: 500},
					{RuleID: "fn_length", Message: "long body"},
				},
			}},
		}},
	}}
}

func TestSearchFiltersAndRankedIndexes(t *testing.T) {
	tests := []struct {
		name  string
		query SearchQuery
		want  []string
	}{
		{"browse", SearchQuery{}, []string{"Branchy", "Simple", "Calm"}},
		{"name", SearchQuery{Query: "BRANCH"}, []string{"Branchy"}},
		{"source", SearchQuery{Query: "STORE.save"}, []string{"Simple"}},
		{"package", SearchQuery{Query: "EXAMPLE"}, []string{"Branchy", "Simple"}},
		{"path", SearchQuery{Query: "CALM.GO"}, []string{"Calm"}},
		{"finding message", SearchQuery{Query: "io BOUNDARY"}, []string{"Branchy"}},
		{"rule text", SearchQuery{Query: "SIDE_EFFECT"}, []string{"Branchy"}},
		{"findings", SearchQuery{FindingsOnly: true}, []string{"Branchy"}},
		{"rule", SearchQuery{Rule: "fn_length"}, []string{"Branchy"}},
		{"unknown rule", SearchQuery{Rule: "no_such_rule"}, []string{}},
		{"combined", SearchQuery{Path: "HOT", MinComplexity: 2, MinCognitiveComplexity: 8}, []string{"Branchy"}},
		{"AND", SearchQuery{Path: "calm", FindingsOnly: true}, []string{}},
		{"literal", SearchQuery{Query: ".*"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.query.Limit = 20
			report := searchFixture()
			result, err := SearchReport(report, tt.query)
			if err != nil {
				t.Fatal(err)
			}
			names := []string{}
			ranked := rankReport(report)
			for _, match := range result.Matches {
				names = append(names, match.Name)
				if got := ranked.Files[match.FileIndex].Functions[match.FunctionIndex].Name; got != match.Name {
					t.Fatalf("index addresses %s, want %s", got, match.Name)
				}
			}
			if !reflect.DeepEqual(names, tt.want) || result.Total != len(tt.want) {
				t.Fatalf("names=%v total=%d, want %v", names, result.Total, tt.want)
			}
			if !reflect.DeepEqual(report, searchFixture()) {
				t.Fatal("search mutates the report")
			}
		})
	}
}

func TestSearchPagesAndEmptyResults(t *testing.T) {
	for _, offset := range []int{0, 1, 2, 3, 4, math.MaxInt} {
		result, err := SearchReport(searchFixture(), SearchQuery{Offset: offset, Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 3 || result.Offset != offset || result.Matches == nil {
			t.Fatalf("invalid page: %+v", result)
		}
		if offset < 2 {
			if result.NextOffset == nil || *result.NextOffset != offset+1 {
				t.Fatalf("next offset: %+v", result)
			}
		} else if result.NextOffset != nil {
			t.Fatalf("unexpected next offset: %+v", result)
		}
		if offset >= 3 && len(result.Matches) != 0 {
			t.Fatal("page past end has matches")
		}
	}
}

func TestSearchRuleRestrictsReturnedFindingsAndPreservesFailureStatus(t *testing.T) {
	report := searchFixture()
	report.Quality = &domain.QualityAnalysis{Status: "error", Error: "type error"}
	result, err := SearchReport(report, SearchQuery{Rule: "fn_length", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 1 || len(result.Matches[0].Findings) != 1 || result.Matches[0].Findings[0].RuleID != "fn_length" {
		t.Fatalf("findings: %+v", result)
	}
	if result.Quality.Status != "error" || result.Quality.Error != "type error" {
		t.Fatal("quality failure lost")
	}
}

func TestSearchRejectsInvalidBounds(t *testing.T) {
	for _, query := range []SearchQuery{{Limit: 0}, {Limit: -1}, {Limit: 201}, {Limit: 20, Offset: -1}, {Limit: 20, MinComplexity: -1}, {Limit: 20, MinCognitiveComplexity: -1}} {
		if _, err := SearchReport(domain.Report{}, query); err == nil {
			t.Fatalf("accepted %+v", query)
		}
	}
}
