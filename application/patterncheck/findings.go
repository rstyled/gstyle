package patterncheck

import (
	"slices"

	"github.com/rstyled/gstyle/domain/patterns"
)

// Matching groups are search inputs, not findings. A shared-helper proposal
// must pass the source verifier before it is published.
func publishFindings(report *patterns.PatternsReport) {
	report.Candidates = slices.DeleteFunc(report.Candidates, patterns.UnsupportedAbstraction)
	report.SignatureGroups = nil
	report.Suppressed = nil
}
