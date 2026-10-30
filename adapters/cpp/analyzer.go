package cpp

import (
	"context"
	"fmt"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/quality"
)

type Analyzer struct {
	Context    context.Context
	SourceRoot string
	Config     *quality.Config
	session    *extractionSession
}

func Extract(ctx context.Context, paths []string) (*Extraction, error) {
	return ExtractFrom(ctx, paths, "")
}

func ExtractFrom(ctx context.Context, paths []string, sourceRoot string) (*Extraction, error) {
	if len(paths) != 1 {
		return nil, fmt.Errorf("--cpg requires exactly one C++ CPG GraphSON file or export directory")
	}
	return LoadFrom(ctx, paths[0], sourceRoot)
}

func (a Analyzer) Analyze(paths []string) (domain.Report, error) {
	ctx := a.Context
	if ctx == nil {
		ctx = context.Background()
	}
	key := extractionKey(paths, a.SourceRoot)
	a.session.store(key, nil)
	extracted, err := ExtractFrom(ctx, paths, a.SourceRoot)
	if err != nil {
		return domain.Report{}, err
	}
	a.session.store(key, extracted)
	config := DefaultQualityConfig()
	if a.Config != nil {
		config = *a.Config
	}
	return extracted.ReportWithConfig(config), nil
}

func (a Analyzer) Patterns(ctx context.Context, paths []string) (*patterns.PatternsReport, error) {
	return a.PatternsWithConfig(ctx, paths, patterns.DefaultConfig())
}

func (a Analyzer) PatternsWithConfig(ctx context.Context, paths []string, config patterns.Config) (*patterns.PatternsReport, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	extracted, err := a.patternExtraction(ctx, paths)
	if err != nil {
		return nil, err
	}
	return extracted.PatternsWithConfig(config, nil), nil
}

func (e *Extraction) Report() domain.Report {
	return e.ReportWithConfig(DefaultQualityConfig())
}

func (e *Extraction) ReportWithConfig(config quality.Config) domain.Report {
	report := domain.Report{Root: e.Root, Files: []domain.File{}}
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
	e.attachQuality(&report, config)
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
