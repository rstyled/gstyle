package patterncheck

import (
	"github.com/rstyled/gstyle/adapters/sarif"
	"github.com/rstyled/gstyle/domain/patterns"
	"io"
	"strings"
)

func writeSARIF(output io.Writer, report *patterns.PatternsReport) error {
	findings := make([]sarif.Finding, 0, len(report.Candidates))
	for _, candidate := range report.Candidates {
		finding := sarif.Finding{RuleID: string(candidate.Kind), Message: strings.Join([]string{candidate.Observation, candidate.Inference, candidate.PossibleRefactor}, "\n")}
		for _, site := range candidate.Sites {
			finding.Locations = append(finding.Locations, sarif.Location{Path: site.Path, Line: site.Line})
		}
		findings = append(findings, finding)
	}
	return sarif.Write(output, findings)
}
