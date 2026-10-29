package searchcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/rstyled/gstyle/adapters/gocyclo"
	"github.com/rstyled/gstyle/adapters/goquality"
	"github.com/rstyled/gstyle/adapters/joern"
	"github.com/rstyled/gstyle/application"
	"github.com/rstyled/gstyle/application/qualitycheck"
)

type options struct {
	cpg                  bool
	query                application.SearchQuery
	config, format, tags string
	tests                bool
}

func NewCommand() (*flag.FlagSet, func(context.Context, []string, io.Writer) error) {
	opts := options{}
	flags := flag.NewFlagSet("search", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&opts.cpg, "cpg", false, "read a Joern full GraphSON export instead of Go source")
	flags.StringVar(&opts.query.Query, "query", "", "literal search in path, package, function, source, and finding text (case-insensitive)")
	flags.StringVar(&opts.query.Path, "path", "", "literal path substring (case-insensitive)")
	flags.StringVar(&opts.query.Rule, "rule", "", "exact quality rule ID; implies findings-only")
	flags.BoolVar(&opts.query.FindingsOnly, "findings-only", false, "only functions with quality findings")
	flags.IntVar(&opts.query.MinComplexity, "min-complexity", 0, "minimum cyclomatic complexity")
	flags.IntVar(&opts.query.MinCognitiveComplexity, "min-cognitive-complexity", 0, "minimum cognitive complexity")
	flags.IntVar(&opts.query.Offset, "offset", 0, "number of matching functions to skip")
	flags.IntVar(&opts.query.Limit, "limit", 20, "maximum matching functions (1-200)")
	flags.StringVar(&opts.config, "config", "", "TOML quality policy")
	flags.StringVar(&opts.format, "format", "json", "output format: json or text")
	flags.StringVar(&opts.tags, "tags", "", "Go build tags for typed quality analysis")
	flags.BoolVar(&opts.tests, "tests", true, "include tests in typed quality analysis (syntax scan always includes them)")
	return flags, func(ctx context.Context, _ []string, output io.Writer) error {
		if !flags.Parsed() {
			return errors.New("search flags must be parsed before execution")
		}
		return opts.run(ctx, flags.Args(), output)
	}
}

func (opts options) run(ctx context.Context, paths []string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := opts.validate(); err != nil {
		return err
	}
	if opts.cpg {
		return opts.execute(ctx, joern.Analyzer{Context: ctx}, paths, output)
	}
	return opts.runGo(ctx, paths, output)
}

func (opts options) runGo(ctx context.Context, paths []string, output io.Writer) error {
	config, err := qualitycheck.LoadConfig(opts.config)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	analyzer := goquality.ReportAnalyzer{Complexity: gocyclo.NewAnalyzer(), Config: config, Tests: opts.tests}
	if opts.tags != "" {
		analyzer.BuildFlags = []string{"-tags=" + opts.tags}
	}
	return opts.execute(ctx, analyzer, paths, output)
}

func (opts options) validate() error {
	if opts.invalidCPGOptions() {
		return fmt.Errorf("--cpg does not support Go build tags or typed quality policies")
	}
	if err := opts.query.Validate(); err != nil {
		return err
	}
	if opts.format != "json" && opts.format != "text" {
		return fmt.Errorf("format must be json or text")
	}
	return nil
}

func (opts options) invalidCPGOptions() bool {
	return opts.cpg && (opts.tags != "" || opts.config != "")
}

func (opts options) execute(ctx context.Context, analyzer application.Analyzer, paths []string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	report, err := analyzer.Analyze(paths)
	if err != nil {
		return err
	}
	result, err := application.SearchReport(report, opts.query)
	if err != nil {
		return err
	}
	if err := writeResult(output, result, opts.format); err != nil {
		return err
	}
	if report.Quality.Status == "error" {
		return errors.New("quality analysis failed: " + report.Quality.Error)
	}
	return ctx.Err()
}

func writeResult(output io.Writer, result application.SearchResult, format string) error {
	data, err := encodeResult(result, format)
	if err != nil {
		return err
	}
	page := bytes.NewReader(data)
	_, err = io.Copy(output, page)
	return err
}

func encodeResult(result application.SearchResult, format string) ([]byte, error) {
	if format == "json" {
		data, err := json.Marshal(result)
		return append(data, '\n'), err
	}
	return []byte(resultText(result)), nil
}

func resultText(result application.SearchResult) string {
	var lines []string
	for _, match := range result.Matches {
		lines = append(lines, fmt.Sprintf("%s:%d:%d %s CC=%d COG=%d", match.Path, match.Line, match.Column, match.Name, match.Complexity, match.CognitiveComplexity))
		for _, finding := range match.Findings {
			lines = append(lines, fmt.Sprintf("  %s: %s", finding.RuleID, finding.Message))
		}
	}
	footer := fmt.Sprintf("%d matching functions; offset %d; returned %d\n", result.Total, result.Offset, len(result.Matches))
	if result.NextOffset != nil {
		footer += fmt.Sprintf("next offset: %d\n", *result.NextOffset)
	}
	if result.Quality != nil && result.Quality.Status == "unavailable" {
		footer += "quality unavailable: " + result.Quality.Error + "\n"
	}
	return strings.Join(append(lines, footer), "\n")
}
