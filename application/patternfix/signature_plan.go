package patternfix

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/patterns"
)

var errOverlappingPlans = errors.New("overlapping fix plans")

func signatureSpecs(specs []*patterns.FixSpec) ([]*patterns.FixSpec, []*patterns.FixSpec) {
	var signature, local []*patterns.FixSpec
	for _, spec := range specs {
		if multiFileFix(spec) {
			signature = append(signature, spec)
		} else {
			local = append(local, spec)
		}
	}
	return signature, local
}

func addSignaturePlans(plan *fixPlan, specs []*patterns.FixSpec) error {
	for _, spec := range specs {
		if err := addSignaturePlan(plan, spec); err != nil && !recordSignatureSkip(plan, spec, err) {
			return fmt.Errorf("plan %s: %w", spec.Kind, err)
		}
	}
	return nil
}

func addSignaturePlan(plan *fixPlan, spec *patterns.FixSpec) error {
	if spec.Context != plan.sources {
		return fmt.Errorf("fixes use different extraction revisions")
	}
	overlay, err := multiFileOverlay(spec)
	if err != nil {
		return err
	}
	primary, err := filepath.Abs(spec.File)
	if err != nil {
		return err
	}
	return appendSignatureFiles(plan, overlay, primary)
}

func recordSignatureSkip(plan *fixPlan, spec *patterns.FixSpec, err error) bool {
	action, reason := signatureSkipReason(err)
	if action == "" {
		return false
	}
	plan.skipped = append(plan.skipped, fmt.Sprintf("%s %s in %s:%d: %s", action, spec.Kind, spec.File, spec.Line, reason))
	return true
}

func signatureSkipReason(err error) (string, string) {
	if errors.Is(err, gopatterns.ErrUnsupportedFix) {
		return "skipped", err.Error()
	}
	if errors.Is(err, errOverlappingPlans) {
		return "deferred", "overlaps an earlier fix in this phase"
	}
	return "", ""
}

func appendSignatureFiles(plan *fixPlan, overlay map[string][]byte, primary string) error {
	var files []filePlan
	primary = signatureCountPath(overlay, primary)
	for path, after := range overlay {
		if planContains(plan.files, path) {
			return fmt.Errorf("%w: %s", errOverlappingPlans, path)
		}
		snapshot, ok := plan.sources.Snapshot(plan.sources.Reference(path))
		if !ok {
			return fmt.Errorf("source evidence is missing: %s", path)
		}
		count := 0
		if path == primary {
			count = 1
		}
		files = append(files, filePlan{path: path, before: []byte(snapshot.Content), after: after, count: count})
	}
	plan.files = append(plan.files, files...)
	return nil
}

func planContains(files []filePlan, path string) bool {
	for _, file := range files {
		if file.path == path {
			return true
		}
	}
	return false
}

func reportSkippedPlans(plan fixPlan, output io.Writer) error {
	for _, message := range plan.skipped {
		if _, err := fmt.Fprintln(output, message); err != nil {
			return err
		}
	}
	return nil
}

// A type-only declaration file can remain unchanged while its operations move.
func signatureCountPath(overlay map[string][]byte, primary string) string {
	if _, ok := overlay[primary]; ok {
		return primary
	}
	var first string
	for path := range overlay {
		if first == "" || path < first {
			first = path
		}
	}
	return first
}

func multiFileFix(spec *patterns.FixSpec) bool {
	if spec.Kind == patterns.Parameterize {
		return spec.Params["secondary_file"] != ""
	}
	return spec.Kind == patterns.ValueObject || spec.Kind == patterns.AnemicModel
}

func multiFileOverlay(spec *patterns.FixSpec) (map[string][]byte, error) {
	if spec.Kind == patterns.Parameterize {
		return gopatterns.PlanParameterizationFix(spec)
	}
	return gopatterns.PlanSignatureFix(spec)
}
