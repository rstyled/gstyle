package cpp

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
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
		fact := function.patternFact(id)
		facts = append(facts, fact)
		for _, hit := range function.source.emptyCatches {
			hit.FunctionID = id
			options.EmptyCatches = append(options.EmptyCatches, hit)
		}
	}
	report := patterns.RunCPP(facts, options)
	return &report
}

func (function Function) patternFact(id string) *patterns.MiningFacts {
	fact := patterns.MiningFacts{}
	if function.mining != nil {
		fact = *function.mining
	}
	fact.ID, fact.Name, fact.Path, fact.Line, fact.EndLine = id, function.Score.Name, function.Path, function.Score.Line, function.Score.EndLine
	fact.Canonical, fact.GuardClauses, fact.ConstantBoolReturns = function.Graph, function.source.guards, function.source.booleans
	for _, effect := range function.Quality.Effects {
		fact.EffectKinds = append(fact.EffectKinds, string(effect.Kind))
	}
	if len(function.Quality.Calls) != 0 {
		fact.EffectKinds = append(fact.EffectKinds, string(quality.UnknownEffect))
	}
	for _, mutation := range function.Quality.Mutations {
		if mutation.Provenance != quality.Local {
			fact.Mutates = true
		}
	}
	return &fact
}
