package cpp

import (
	"github.com/rstyled/gstyle/domain/patterns"
)

const patternScope = "gstyle.cpp.patterns/4: shared analyses over verified CPG facts; dependency sizes are measurements, not refactor findings; candidates require source review"

func (e *Extraction) Patterns() *patterns.PatternsReport {
	return e.PatternsWithConfig(patterns.DefaultConfig(), nil)
}

func (e *Extraction) PatternsWithMining(params patterns.Params, cache *patterns.WlCache) *patterns.PatternsReport {
	return e.patterns(params, patterns.DefaultConfig().Measurements, cache)
}

func (e *Extraction) PatternsWithConfig(config patterns.Config, cache *patterns.WlCache) *patterns.PatternsReport {
	report := e.patterns(config.MiningParams(), config.Measurements, cache)
	report.Configuration = &config
	return report
}

func (e *Extraction) patterns(params patterns.Params, limits patterns.MeasurementLimits, cache *patterns.WlCache) *patterns.PatternsReport {
	facts := make([]*patterns.MiningFacts, 0, len(e.Functions))
	options := patterns.CPPOptions{Reason: patternScope, Source: e.sourceOptions, Params: params, WlCache: cache, Measurements: limits}
	for _, function := range e.Functions {
		graph := function.Graph
		id := graph.Tables.Text(graph.Function.ID)
		facts = append(facts, function.mining)
		for _, hit := range function.source.emptyCatches {
			hit.FunctionID = id
			options.EmptyCatches = append(options.EmptyCatches, hit)
		}
	}
	report := patterns.RunCPP(facts, options)
	return &report
}
