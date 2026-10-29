package patterncheck

import (
	"context"
	"io"

	"github.com/rstyled/gstyle/adapters/joern"
)

func runCPG(ctx context.Context, opts options, output io.Writer) error {
	extracted, err := joern.Extract(ctx, opts.paths)
	if err != nil {
		return err
	}
	if opts.format == "pdg-json" {
		return writePDGArray(ctx, output, extracted.Graphs())
	}
	return writeReport(output, extracted.Patterns(), opts.format)
}
