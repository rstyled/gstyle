package gopatterns

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

// ErrUnsupportedFix marks a known limit of the rewrite profile. Missing
// evidence, invalid plans, IO failures and failed type checks remain errors.
var ErrUnsupportedFix = errors.New("unsupported fix")

func unsupportedFix(format string, values ...any) error {
	return fmt.Errorf("%w: %s", ErrUnsupportedFix, fmt.Sprintf(format, values...))
}

func checkSignatureTargetEvidence(context *pdg.SourceContext, targets []string) error {
	required := make(map[string]bool, len(targets))
	for _, target := range targets {
		required[target] = true
	}
	for _, function := range context.Functions {
		delete(required, function.ID)
	}
	for _, target := range targets {
		if required[target] {
			return fmt.Errorf("signature target evidence is missing: %s", target)
		}
	}
	return nil
}

func signatureSourceRef(context *pdg.SourceContext, spec *patterns.FixSpec) (pdg.Ref, error) {
	path, err := filepath.Abs(spec.File)
	if err != nil {
		return 0, err
	}
	ref := context.Reference(path)
	if ref == 0 {
		return 0, fmt.Errorf("source evidence is missing: %s", path)
	}
	if spec.Kind != patterns.ValueObject {
		return ref, nil
	}
	return ref, checkSignatureTargetEvidence(context, spec.Targets)
}
