package patterncheck

import (
	"context"
	"fmt"
	"io"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/adapters/pdgjson"
	"github.com/rstyled/gstyle/domain/pdg"
)

// exportPDGs bypasses mining and its caches. The array contains one compat
// document per canonical callable; documents are expanded and encoded one at a time.
func exportPDGs(ctx context.Context, opts options, output io.Writer) error {
	ext, err := gopatterns.Extract(ctx, "", opts.paths)
	if err != nil {
		return err
	}
	if err := validateExport(ext.Callables); err != nil {
		return err
	}
	return writePDGArray(ctx, output, ext.Callables)
}

// Reject invalid native input before writing any export bytes.
func validateExport(graphs []*pdg.Graph) error {
	for i, graph := range graphs {
		if err := pdg.Validate(graph); err != nil {
			return fmt.Errorf("export graph %d: %w", i, err)
		}
	}
	return nil
}

func writePDGArray(ctx context.Context, output io.Writer, graphs []*pdg.Graph) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := io.WriteString(output, "[\n"); err != nil {
		return err
	}
	for i, graph := range graphs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := exportFunction(output, graph, i); err != nil {
			return err
		}
	}
	_, err := io.WriteString(output, "]\n")
	return err
}
func exportFunction(output io.Writer, graph *pdg.Graph, index int) error {
	if index != 0 {
		if _, err := io.WriteString(output, ",\n"); err != nil {
			return err
		}
	}
	if err := pdgjson.Write(output, graph); err != nil {
		return fmt.Errorf("export graph %d: %w", index, err)
	}
	return nil
}
