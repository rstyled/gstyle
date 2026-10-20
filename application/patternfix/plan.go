package patternfix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

type filePlan struct {
	path          string
	before, after []byte
	count         int
}

type fixPlan struct {
	skipped []string
	sources *pdg.SourceContext
	files   []filePlan
}

func (p fixPlan) overlay() map[string][]byte {
	overlay := make(map[string][]byte, len(p.files))
	for _, file := range p.files {
		overlay[file.path] = file.after
	}
	return overlay
}

func buildFixPlan(ctx context.Context, specs []*patterns.FixSpec) (fixPlan, error) {
	plan := fixPlan{}
	if len(specs) == 0 {
		return plan, nil
	}
	plan.sources = specs[0].Context
	if err := gopatterns.CheckFixSources(plan.sources); err != nil {
		return plan, err
	}
	signature, local := signatureSpecs(specs)
	byFile := groupByFile(local)
	paths := sortedPlanPaths(byFile)
	files, err := planLocalFiles(ctx, paths, byFile, plan.sources)
	if err != nil {
		return plan, err
	}
	plan.files = files
	if err := addSignaturePlans(&plan, signature); err != nil {
		return plan, err
	}
	sort.Slice(plan.files, func(i, j int) bool { return plan.files[i].path < plan.files[j].path })
	return plan, nil
}

func sortedPlanPaths(byFile map[string][]*patterns.FixSpec) []string {
	paths := make([]string, 0, len(byFile))
	for path := range byFile {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func planFile(path string, specs []*patterns.FixSpec, sources *pdg.SourceContext) (filePlan, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filePlan{}, err
	}
	snapshot, ok := sources.Snapshot(sources.Reference(absolute))
	if !ok {
		return filePlan{}, fmt.Errorf("source evidence is missing: %s", path)
	}
	if err := samePlanContext(specs, sources); err != nil {
		return filePlan{}, err
	}
	before := []byte(snapshot.Content)
	after, count, skipped, err := gopatterns.PlanLocalFixes(sortSpecsForApply(specs), snapshot)
	for _, reason := range skipped {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", path, reason)
	}
	return filePlan{path: absolute, before: before, after: after, count: count}, err
}

func samePlanContext(specs []*patterns.FixSpec, sources *pdg.SourceContext) error {
	for _, spec := range specs {
		if spec.Context != sources {
			return fmt.Errorf("fixes use different extraction revisions")
		}
	}
	return nil
}

func executeFixPlan(ctx context.Context, plan fixPlan, opts options, output io.Writer) (int, []string, error) {
	if len(plan.files) == 0 {
		return 0, nil, nil
	}
	if err := validateFixPlan(ctx, plan); err != nil {
		return 0, nil, err
	}
	if opts.apply {
		return writeFixPlan(plan, output)
	}
	var count int
	for _, file := range plan.files {
		if err := showDiff(file.path, file.before, file.after, output); err != nil {
			return 0, nil, err
		}
		count += file.count
	}
	return count, nil, nil
}

// All semantic validation finishes before the first write. Files retain their
// permissions. A write failure restores files already written in this batch.
func writeFixPlan(plan fixPlan, output io.Writer) (int, []string, error) {
	if err := gopatterns.CheckFixSources(plan.sources); err != nil {
		return 0, nil, err
	}
	var count int
	var written []string
	for i, file := range plan.files {
		if err := writePlanFile(file); err != nil {
			return 0, nil, rollbackPlan(plan.files[:i], err)
		}
		count += file.count
		written = append(written, file.path)
	}
	for _, file := range plan.files {
		if err := reportPlanFile(file, output); err != nil {
			return count, written, err
		}
	}
	return count, written, nil
}

func writePlanFile(file filePlan) error {
	info, err := os.Lstat(file.path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("fix target is not a regular file: %s", file.path)
	}
	staged, err := os.CreateTemp(filepath.Dir(file.path), ".gstyle-fix-*")
	if err != nil {
		return err
	}
	defer os.Remove(staged.Name())
	defer staged.Close()
	if err := stagePlanFile(staged, file.after, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(staged.Name(), file.path)
}

func rollbackPlan(files []filePlan, cause error) error {
	failures := []error{cause}
	for _, file := range files {
		file.after = file.before
		if err := writePlanFile(file); err != nil {
			failures = append(failures, fmt.Errorf("rollback %s fails: %w", file.path, err))
		}
	}
	return errors.Join(failures...)
}

func planLocalFiles(ctx context.Context, paths []string, byFile map[string][]*patterns.FixSpec, sources *pdg.SourceContext) ([]filePlan, error) {
	var files []filePlan
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		file, err := planFile(path, byFile[path], sources)
		if err != nil {
			return nil, err
		}
		if file.count > 0 {
			files = append(files, file)
		}
	}
	return files, nil
}

func stagePlanFile(file *os.File, content []byte, mode os.FileMode) error {
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func validateFixPlan(ctx context.Context, plan fixPlan) error {
	if err := gopatterns.ValidateFixOverlay(ctx, plan.sources, plan.overlay()); err != nil {
		return err
	}
	return ctx.Err()
}

func reportPlanFile(file filePlan, output io.Writer) error {
	if file.count == 0 {
		_, err := fmt.Fprintf(output, "updated callers in %s\n", file.path)
		return err
	}
	_, err := fmt.Fprintf(output, "fixed %d candidate(s) in %s\n", file.count, file.path)
	return err
}
