package nextscore

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rstyled/gstyle/application/patterncheck"
	"github.com/rstyled/gstyle/domain/patterns"
)

// Run implements `gstyle next`: show the single highest-impact pattern
// candidate with fix instructions. Agent-focused: one thing, not a firehose.
func Run(ctx context.Context, args []string, output io.Writer) error {
	report, err := patterncheck.GetReport(ctx, args)
	if err != nil {
		return err
	}
	_, err = io.WriteString(output, nextText(report))
	return err
}

func nextText(report *patterns.PatternsReport) string {
	var text strings.Builder
	output := &text
	if len(report.Candidates) == 0 {
		fmt.Fprintln(output, "No pattern candidates.")
		writeAnalysisCoverage(output, report.AnalysisCoverage)
		return text.String()
	}
	writeAnalysisCoverage(output, report.AnalysisCoverage)
	top := report.Candidates[0]
	fmt.Fprintf(output, "Next: %s (score %.2f)\n", top.Kind, float64(top.ScoreMilli)/1000)
	fmt.Fprintf(output, "  %s\n", top.Observation)
	fmt.Fprintf(output, "  %s\n", top.Inference)
	for _, s := range top.Sites {
		fmt.Fprintf(output, "  %s:%d %s\n", s.Path, s.Line, s.Name)
	}
	writeFixInstructions(output, top)
	return text.String()
}

// Score implements `gstyle score`: a single 0-100 north-star number.
// 100 = no scored findings; it does not establish complete analysis coverage.
func Score(ctx context.Context, args []string, output io.Writer) error {
	report, err := patterncheck.GetReport(ctx, args)
	if err != nil {
		return err
	}
	_, err = io.WriteString(output, scoreText(report))
	return err
}

func scoreText(report *patterns.PatternsReport) string {
	var text strings.Builder
	output := &text
	score := computeScore(report)
	fmt.Fprintf(output, "Score: %d/100\n", score)
	fmt.Fprintf(output, "  %d active candidates", len(report.Candidates))
	if len(report.Suppressed) > 0 {
		fmt.Fprintf(output, ", %d suppressed", len(report.Suppressed))
	}
	fmt.Fprintln(output)
	writeAnalysisCoverage(output, report.AnalysisCoverage)
	return text.String()
}

// computeScore maps candidates to 0-100. Each candidate deducts points
// weighted by severity (score). Starts at 100, floors at 0.
func computeScore(report *patterns.PatternsReport) int {
	score := 100
	for _, c := range report.Candidates {
		// Severity weight: score 0.8 deducts 8 points, 0.4 deducts 4, etc.
		deduction := int(float64(c.ScoreMilli) / 100)
		score -= deduction
	}
	if score < 0 {
		score = 0
	}
	return score
}

func writeAnalysisCoverage(output *strings.Builder, coverage []patterns.AnalysisCoverage) {
	if !patterns.HasIncompleteAnalysis(coverage) {
		return
	}
	fmt.Fprintln(output, "Dependency analysis coverage is incomplete; see gstyle patterns for details.")
}

func writeFixInstructions(output *strings.Builder, candidate patterns.Candidate) {
	if candidate.FixSpec == nil {
		fmt.Fprintln(output, "\nAdvisory finding: no automatic fix is available.")
		return
	}
	fmt.Fprintf(output, "\nFix with: gstyle fix --kind %s %s\n", candidate.FixSpec.Kind, candidate.FixSpec.File)
}
