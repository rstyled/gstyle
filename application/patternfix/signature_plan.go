package patternfix

import (
	"fmt"
	"path/filepath"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/patterns"
)

func signatureSpecs(specs []*patterns.FixSpec) ([]*patterns.FixSpec, []*patterns.FixSpec) {
	var signature, local []*patterns.FixSpec
	for _, spec := range specs {
		if spec.Kind == patterns.ValueObject || spec.Kind == patterns.AnemicModel {
			signature = append(signature, spec)
		} else {
			local = append(local, spec)
		}
	}
	return signature, local
}

func addSignaturePlans(plan *fixPlan, specs []*patterns.FixSpec) error {
	for _, spec := range specs {
		if spec.Context != plan.sources {
			return fmt.Errorf("fixes use different extraction revisions")
		}
		overlay, err := gopatterns.PlanSignatureFix(spec)
		if err != nil {
			return fmt.Errorf("plan %s: %w", spec.Kind, err)
		}
		primary, err := filepath.Abs(spec.File)
		if err != nil {
			return err
		}
		if err := appendSignatureFiles(plan, overlay, primary); err != nil {
			return err
		}
	}
	return nil
}

func appendSignatureFiles(plan *fixPlan, overlay map[string][]byte, primary string) error {
	for path, after := range overlay {
		if planContains(plan.files, path) {
			return fmt.Errorf("fix plans overlap in %s; use separate phases", path)
		}
		snapshot, ok := plan.sources.Snapshot(plan.sources.Reference(path))
		if !ok {
			return fmt.Errorf("source evidence is missing: %s", path)
		}
		count := 0
		if path == primary {
			count = 1
		}
		plan.files = append(plan.files, filePlan{path: path, before: []byte(snapshot.Content), after: after, count: count})
	}
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
