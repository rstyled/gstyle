package changerisk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestRiskCheckAllowsOnlyLowAndPreservesTheReport(t *testing.T) {
	for _, tc := range []struct {
		name, policy, before, after string
		want                        changerisk.Level
	}{
		{"low", "[risk]\npdg_change='low'\n", "func Count() int { return 1 }", "func Count() int { return 2 }", changerisk.Low},
		{"small change", "[risk]\npdg_change='medium'\npdg_low_risk_max_functions=3\n", "func Count() int { return 1 }", "func Count() int { return 2 }", changerisk.Low},
		{"small unresolved change", "[risk]\npdg_change='medium'\npdg_low_risk_max_functions=3\n", "func Count(next func() int) int { return next()+1 }", "func Count(next func() int) int { return next()+2 }", changerisk.Unknown},
		{"medium", "[risk]\npdg_change='medium'\n", "func Count() int { return 1 }", "func Count() int { return 2 }", changerisk.Medium},
		{"high", "[risk]\npdg_change='high'\n", "func Count() int { return 1 }", "func Count() int { return 2 }", changerisk.High},
		{"unrated", "", "func Count() int { return 1 }", "func Count() int { return 2 }", changerisk.Unrated},
		{"unknown", "[risk]\npdg_change='low'\n", "func Count(next func() int) int { return next()+1 }", "func Count(next func() int) int { return next()+2 }", changerisk.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := repository(t)
			write(t, root, "gstyle.toml", tc.policy)
			write(t, root, "source.go", "package sample\n"+tc.before+"\n")
			base := commit(t, root)
			write(t, root, "source.go", "package sample\n"+tc.after+"\n")
			head := commit(t, root)
			t.Chdir(root)
			var output bytes.Buffer
			err := Run(context.Background(), []string{"--base", base, "--head", head, "--check", "--format", "json"}, &output)
			if errors.Is(err, ErrReviewRequired) != (tc.want != changerisk.Low) || (err != nil && !errors.Is(err, ErrReviewRequired)) {
				t.Fatalf("unexpected gate result: %v\n%s", err, output.String())
			}
			var report changerisk.Report
			if err := json.Unmarshal(output.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Risk != tc.want || report.Base != base || report.Head != head {
				t.Fatalf("gate changed or lost its evidence: %+v", report)
			}
			output.Reset()
			if err := Run(context.Background(), []string{"--base", base, "--head", head}, &output); err != nil {
				t.Fatalf("advisory reports must remain successful: %v", err)
			}
		})
	}
}

func TestRiskCheckRequiresAnExplicitBase(t *testing.T) {
	var output bytes.Buffer
	err := Run(context.Background(), []string{"--check"}, &output)
	if err == nil || !strings.Contains(err.Error(), "explicit --base") || output.Len() != 0 {
		t.Fatalf("gate accepted an implicit empty comparison: %v %s", err, output.String())
	}
}

func TestRiskCheckRejectsEmbeddedDocumentationChanges(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "[risk]\npdg_change='low'\ndependency_change='low'\n")
	write(t, root, "source.go", "package sample\nimport _ \"embed\"\n//go:embed README.md\nvar Text string\n")
	write(t, root, "README.md", "before\n")
	base := commit(t, root)
	write(t, root, "README.md", "after\n")
	head := commit(t, root)
	t.Chdir(root)
	var output bytes.Buffer
	err := Run(context.Background(), []string{"--base", base, "--head", head, "--check", "--format", "json"}, &output)
	if !errors.Is(err, ErrReviewRequired) {
		t.Fatalf("embedded document passed the automation gate: %v\n%s", err, output.String())
	}
	var report changerisk.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Risk != changerisk.Unknown || !strings.Contains(output.String(), "embedded resources") {
		t.Fatalf("embedded resource evidence missing: %s", output.String())
	}
}
