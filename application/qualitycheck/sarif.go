package qualitycheck

import (
	"github.com/rstyled/gstyle/adapters/sarif"
	"github.com/rstyled/gstyle/domain/quality"
	"io"
)

func writeSARIF(output io.Writer, report quality.Report) error {
	findings := make([]sarif.Finding, 0, len(report.Diagnostics))
	for _, diagnostic := range report.Diagnostics {
		findings = append(findings, sarif.Finding{
			RuleID: diagnostic.RuleID, Message: diagnostic.Message,
			Locations: []sarif.Location{{Path: diagnostic.Path, Line: diagnostic.Line}},
		})
	}
	return sarif.Write(output, findings)
}
