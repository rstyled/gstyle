package joern

import (
	"context"
	"fmt"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

const importScope = "gstyle.joern.complexity/1: method-local source complexity and resolved recursion cycles; advisory CPG dependencies; typed mutation/effect checks and source fixes are unavailable"

type Analyzer struct{ Context context.Context }

func Extract(ctx context.Context, paths []string) (*Extraction, error) {
	if len(paths) != 1 {
		return nil, fmt.Errorf("--cpg requires exactly one Joern GraphSON file or export directory")
	}
	return Load(ctx, paths[0])
}

func (a Analyzer) Analyze(paths []string) (domain.Report, error) {
	ctx := a.Context
	if ctx == nil {
		ctx = context.Background()
	}
	extracted, err := Extract(ctx, paths)
	if err != nil {
		return domain.Report{}, err
	}
	return extracted.Report(), nil
}

func (a Analyzer) Patterns(ctx context.Context, paths []string) (*patterns.PatternsReport, error) {
	extracted, err := Extract(ctx, paths)
	if err != nil {
		return nil, err
	}
	return extracted.Patterns(), nil
}

func (e *Extraction) Report() domain.Report {
	report := domain.Report{Root: e.Root, Files: []domain.File{}, Quality: &domain.QualityAnalysis{Status: "unavailable", Error: importScope}}
	files := map[string]int{}
	for _, function := range e.Functions {
		index, exists := files[function.Path]
		if !exists {
			index = len(report.Files)
			files[function.Path] = index
			report.Files = append(report.Files, domain.File{Path: function.Path})
		}
		appendFunction(&report.Files[index], function.Score)
		report.Functions++
		report.Total += function.Score.Complexity
		report.CognitiveTotal += function.Score.CognitiveComplexity
	}
	report.Average, report.CognitiveAverage = average(report.Total, report.Functions), average(report.CognitiveTotal, report.Functions)
	return report
}

func appendFunction(file *domain.File, function domain.Function) {
	file.Functions = append(file.Functions, function)
	file.Total += function.Complexity
	file.CognitiveTotal += function.CognitiveComplexity
	file.Peak, file.CognitivePeak = max(file.Peak, function.Complexity), max(file.CognitivePeak, function.CognitiveComplexity)
	file.Average, file.CognitiveAverage = average(file.Total, len(file.Functions)), average(file.CognitiveTotal, len(file.Functions))
}

func average(total, count int) float64 {
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

func (e *Extraction) Graphs() []*pdg.Graph {
	graphs := make([]*pdg.Graph, 0, len(e.Functions))
	for _, function := range e.Functions {
		graphs = append(graphs, function.Graph)
	}
	return graphs
}

func (e *Extraction) Patterns() *patterns.PatternsReport {
	facts := make([]*patterns.MiningFacts, 0, len(e.Functions))
	for _, function := range e.Functions {
		graph := function.Graph
		facts = append(facts, &patterns.MiningFacts{ID: graph.Tables.Text(graph.Function.ID), Name: function.Score.Name, Path: function.Path, Canonical: graph})
	}
	return &patterns.PatternsReport{Candidates: patterns.ThinSliceCandidates(patterns.FindThinSlices(facts), facts), AnalysisCoverage: []patterns.AnalysisCoverage{{Kind: patterns.ThinSliceKind, Partial: len(facts), Reason: importScope}}}
}
