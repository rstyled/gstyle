package patternfix

import (
	"context"
	"fmt"
	"io"
	"maps"
	"sort"

	"github.com/rstyled/gstyle/adapters/gitchanged"
	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Preview runs the same scheduler against cumulative compiler overlays.
// The original disk evidence remains authoritative throughout the run.
type previewRevision struct {
	disk     *pdg.SourceContext
	overlay  map[string][]byte
	original map[string][]byte
}

func (p *previewRevision) advance(ctx context.Context, plan fixPlan) (int, []string, error) {
	if len(plan.files) == 0 {
		return 0, nil, nil
	}
	if err := validateFixPlan(ctx, plan); err != nil {
		return 0, nil, err
	}
	var count int
	var paths []string
	for _, file := range plan.files {
		if _, ok := p.original[file.path]; !ok {
			p.original[file.path] = file.before
		}
		p.overlay[file.path] = file.after
		count += file.count
		paths = append(paths, file.path)
	}
	return count, paths, nil
}

func (p *previewRevision) show(output io.Writer) error {
	paths := make([]string, 0, len(p.overlay))
	for path := range p.overlay {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if _, err := showDiff(path, p.original[path], p.overlay[path], 0, output); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(output, "dry-run; use --apply to write")
	return err
}

func extractFixRevision(ctx context.Context, opts options) (*gopatterns.Extraction, error) {
	if opts.preview == nil {
		return gopatterns.ExtractForFix(ctx, "", opts.paths)
	}
	p := opts.preview
	ext, err := gopatterns.ExtractFixOverlay(ctx, "", opts.paths, gopatterns.SourceOverlay{Files: maps.Clone(p.overlay), Disk: p.disk})
	if err != nil {
		return nil, err
	}
	if p.disk == nil {
		p.disk = ext.Sources
	}
	return ext, nil
}

func runPhasedDryRun(ctx context.Context, opts options, output io.Writer) error {
	preview := &previewRevision{overlay: make(map[string][]byte), original: make(map[string][]byte)}
	opts.preview = preview
	if err := runPhased(ctx, opts, output); err != nil {
		return err
	}
	if err := gopatterns.CheckFixSources(preview.disk); err != nil {
		return err
	}
	return preview.show(output)
}

func fixChangedDiff(cwd string, opts options) (*gitchanged.Diff, error) {
	if opts.preview == nil {
		return gitchanged.NewDiff(cwd, opts.paths, opts.base)
	}
	return gitchanged.NewOverlayDiff(cwd, opts.paths, opts.base, opts.preview.overlay)
}
