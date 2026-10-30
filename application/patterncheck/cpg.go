package patterncheck

import (
	"context"
	"io"

	"github.com/rstyled/gstyle/adapters/cpp"
	"github.com/rstyled/gstyle/domain/patterns"
)

func runCPG(ctx context.Context, opts options, output io.Writer) error {
	extracted, err := cpp.ExtractFrom(ctx, opts.paths, opts.cpgSource)
	if err != nil {
		return err
	}
	if opts.format == "pdg-json" {
		return writePDGArray(ctx, output, extracted.Graphs())
	}
	report, err := mineCPG(extracted, opts)
	if err != nil {
		return err
	}
	return writeReport(output, report, opts.format)
}

func cpgReport(ctx context.Context, opts options) (*patterns.PatternsReport, error) {
	extracted, err := cpp.ExtractFrom(ctx, opts.paths, opts.cpgSource)
	if err != nil {
		return nil, err
	}
	return mineCPG(extracted, opts)
}

func mineCPG(extracted *cpp.Extraction, opts options) (*patterns.PatternsReport, error) {
	cache, save, err := openWlCache(opts.useCache)
	if err != nil {
		return nil, err
	}
	defer save()
	return extracted.PatternsWithConfig(opts.settings, cache), nil
}
