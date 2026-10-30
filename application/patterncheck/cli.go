// Package patterncheck is the IO boundary for gstyle's patterns miner: it
// proposes latent shared abstractions (interfaces, params structs, generics)
// found by structural mining. It is informational only: it always exits 0 on
// success and never feeds the quality gate. Pattern facts live apart from
// quality facts and are never passed to quality.Evaluate.
package patterncheck

import (
	"context"
	"flag"
	"fmt"
	"io"
	"runtime"
	"slices"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/application/qualitycheck"
	"github.com/rstyled/gstyle/domain/patterns"
)

const usage = `Usage: gstyle patterns [OPTIONS] [DIRECTORIES OR GO FILES...]

Mine the codebase for latent shared abstractions: structurally parallel
functions that suggest interfaces, params structs, or generics, plus
inverted conditionals that want to be guard clauses.
Similarity, size and usage statistics alone do not produce findings.
Informational only: always exits 0, never a quality gate.
  --format FORMAT   text (default), json report, sarif, or pdg-json graph export
  --cpg             read one full C++ CPG GraphSON export instead of Go source
  --config PATH     read pattern settings from a gstyle.toml policy
  --threshold N     minimum WL similarity percent (0-100) [default 60]
  --cache           reuse WL refinements from .gstyle/patterns-cache.json,
                    speeding up repeat runs on large repos

Exit 0: always, on success (even with no candidates). Exit 2: extraction or
analysis failure.
`

const wlCachePath = ".gstyle/patterns-cache.json"

type options struct {
	cpg          bool
	cpgSource    string
	format       string
	threshold    uint
	useCache     bool
	paths        []string
	config       string
	settings     patterns.Config
	thresholdSet bool
}

func (opts options) validate() error {
	if err := opts.validateCPGOptions(); err != nil {
		return err
	}
	if !slices.Contains([]string{"text", "json", "sarif", "pdg-json"}, opts.format) {
		return fmt.Errorf("format must be text, json, sarif or pdg-json")
	}
	if opts.format == "pdg-json" && opts.useCache {
		return fmt.Errorf("--cache is not available for pdg-json export")
	}
	if opts.threshold > 100 {
		return fmt.Errorf("threshold must be between 0 and 100 percent")
	}
	return nil
}

func (opts options) validateCPGOptions() error {
	if opts.cpgSource != "" && !opts.cpg {
		return fmt.Errorf("--cpg-source requires --cpg")
	}
	return nil
}

// Run extracts PDGs for paths, mines them for abstraction candidates, and
// writes the report, or exports native graphs when pdg-json is selected.
// It returns nil on success regardless of how many (or
// how few) candidates the miner finds; only extraction, analysis, or IO
// failures are errors.
func Run(ctx context.Context, args []string, output io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.cpg {
		return runCPG(ctx, opts, output)
	}
	if opts.format == "pdg-json" {
		return exportPDGs(ctx, opts, output)
	}
	report, err := GetReport(ctx, args)
	if err != nil {
		return err
	}
	return writeReport(output, report, opts.format)
}

func writeReport(output io.Writer, report *patterns.PatternsReport, format string) error {
	if format == "sarif" {
		return writeSARIF(output, report)
	}
	if format == "json" {
		out, err := patterns.JSON(report)
		if err != nil {
			return err
		}
		_, err = io.WriteString(output, out+"\n")
		return err
	}
	_, err := io.WriteString(output, patterns.Text(report)+"\n")
	return err
}

// GetReport runs the pattern pipeline and returns the report.
// Exported for `gstyle next` and `gstyle score`.
func GetReport(ctx context.Context, args []string) (*patterns.PatternsReport, error) {
	return GetReportWithConfig(ctx, args, nil)
}

func GetReportWithConfig(ctx context.Context, args []string, config *patterns.Config) (*patterns.PatternsReport, error) {
	opts, err := parseOptions(args)
	if err != nil {
		return nil, err
	}
	if config != nil {
		opts = opts.withSettings(*config)
	}
	if err := opts.settings.Validate(); err != nil {
		return nil, err
	}
	return getReport(ctx, opts)
}

func getReport(ctx context.Context, opts options) (*patterns.PatternsReport, error) {
	if opts.cpg {
		return cpgReport(ctx, opts)
	}
	pdgs, err := gopatterns.Extract(ctx, "", opts.paths)
	if err != nil {
		return nil, err
	}
	params := opts.settings.MiningParams()
	cache, saveCache, err := openWlCache(opts.useCache)
	if err != nil {
		return nil, err
	}
	defer saveCache()
	// Extraction's AST and type-checking data are no longer needed. Reclaim
	// that large temporary phase before allocating the mining graphs.
	runtime.GC()
	report := patterns.RunGo(pdgs.PatternFacts(), patterns.Options{
		Callables:         pdgs.Callables,
		DeclarationOrders: pdgs.DeclarationOrders,
		OffsetFallbacks:   pdgs.OffsetFallbacks,
		NilGuards:         pdgs.NilGuards,
		NegativeIndices:   pdgs.NegativeIndices,
		LiteralIndices:    pdgs.LiteralIndices,
		Params:            params,
		AnemicModels:      pdgs.AnemicModels,
		MissingIdentities: pdgs.MissingIdentities,
		DomainServices:    pdgs.DomainServices,
		WlCache:           cache,
	})
	report.Configuration = &opts.settings
	gopatterns.VerifyValueGroups(&report, pdgs)
	if err := gopatterns.VerifyParameterizations(&report, pdgs); err != nil {
		return nil, fmt.Errorf("verify parameterization: %w", err)
	}
	publishFindings(&report)
	return &report, nil
}

// openWlCache loads the WL cache when enabled, returning the cache and a
// deferred save. A missing cache file is fine (cold start); corrupt files
// error out. Save failures are best-effort and never fail the mining run.
func openWlCache(enabled bool) (cache *patterns.WlCache, save func(), err error) {
	save = func() {}
	if !enabled {
		return nil, save, nil
	}
	cache = patterns.NewWlCache()
	if err := cache.Load(wlCachePath); err != nil {
		return nil, save, err
	}
	return cache, func() { _ = cache.Save(wlCachePath) }, nil
}

// Flags returns fresh flag definitions shared by command help and execution.
func Flags() *flag.FlagSet { return flagsFor(&options{}) }

func flagsFor(result *options) *flag.FlagSet {
	flags := flag.NewFlagSet("patterns", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&result.cpg, "cpg", false, "read a full C++ CPG GraphSON export instead of Go source")
	flags.StringVar(&result.cpgSource, "cpg-source", "", "source root for files missing from a C++ export")
	flags.StringVar(&result.config, "config", "", "TOML pattern policy")
	flags.StringVar(&result.format, "format", "text", "output format")
	flags.UintVar(&result.threshold, "threshold", 60, "minimum WL similarity percent (0-100)")
	flags.BoolVar(&result.useCache, "cache", false, "reuse WL refinements from .gstyle/patterns-cache.json")
	return flags
}

func parseOptions(args []string) (options, error) {
	result := options{}
	flags := flagsFor(&result)
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	result.paths = flags.Args()
	flags.Visit(func(flag *flag.Flag) {
		if flag.Name == "threshold" {
			result.thresholdSet = true
		}
	})
	if err := result.validate(); err != nil {
		return options{}, err
	}
	return configureOptions(result)
}

func configureOptions(opts options) (options, error) {
	settings, err := qualitycheck.LoadPatternConfig(opts.config)
	if err != nil {
		return options{}, err
	}
	opts = opts.withSettings(settings)
	return opts, opts.settings.Validate()
}

func (opts options) withSettings(settings patterns.Config) options {
	if opts.thresholdSet {
		settings.SimilarityThreshold = uint32(opts.threshold)
	}
	opts.settings = settings
	return opts
}
