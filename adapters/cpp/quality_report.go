package cpp

import (
	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/quality"
)

func (e *Extraction) attachQuality(report *domain.Report, config quality.Config) {
	facts := e.QualityFacts()
	result, err := quality.Evaluate(facts, config)
	if err != nil {
		report.Quality = &domain.QualityAnalysis{Status: "error", Error: err.Error()}
		return
	}
	report.Quality = &domain.QualityAnalysis{Status: "ready", Report: &result}
	byPosition := qualityPositions(facts, result)
	for i := range report.Files {
		for j := range report.Files[i].Functions {
			function := &report.Files[i].Functions[j]
			location := quality.Location{Path: report.Files[i].Path, Line: function.Line, Column: function.Column, Name: qualityName(function.Name)}
			function.Quality = byPosition[location]
		}
	}
}

func qualityPositions(facts []quality.Function, result quality.Report) map[quality.Location]*domain.FunctionQuality {
	byPosition := map[quality.Location]*domain.FunctionQuality{}
	mutations := map[quality.Location][]quality.Mutation{}
	for _, fact := range facts {
		mutations[fact.Location] = fact.Mutations
	}
	for _, function := range result.Functions {
		byPosition[function.Location] = &domain.FunctionQuality{FunctionResult: function, MutationEvents: mutations[function.Location]}
	}
	for _, diagnostic := range result.Diagnostics {
		function := byPosition[diagnostic.Location]
		function.Diagnostics = append(function.Diagnostics, diagnostic)
	}
	return byPosition
}
