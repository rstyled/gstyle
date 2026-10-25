package changerisk

import (
	"strings"
	"testing"
)

func TestUnratedSummaryExplainsMissingBaseRules(t *testing.T) {
	report := Evaluate(Snapshot{}, Snapshot{}, Config{}, APIReport{})
	if report.PolicyConfigured {
		t.Fatal("empty policy reported as configured")
	}
	if got := ratingSummary(report); got != "No risk rules in `gstyle.toml` at the base commit." {
		t.Fatalf("missing-policy explanation: %s", got)
	}
	report.Policy = "policy/custom.toml"
	if got := Markdown(report); !strings.Contains(got, "No risk rules in `policy/custom.toml` at the base commit.") || !strings.HasPrefix(got, "**Merge risk: unrated** 👀\n") {
		t.Fatalf("custom policy explanation: %s", got)
	}
}

func TestUnratedSummaryNamesOnlyUnconfiguredChanges(t *testing.T) {
	base := Snapshot{Dependencies: []string{"before"}}
	head := Snapshot{Functions: []Function{{Name: "p.F", Body: "new", Graph: Graph{Shape: "new"}}}, Dependencies: []string{"after"}}
	config := Config{Calls: []CallRule{{Callee: "os.WriteFile", Level: High}}}
	report := Evaluate(base, head, config, APIReport{})
	if report.Risk != Unrated || !report.PolicyConfigured {
		t.Fatalf("partial policy evidence lost: %+v", report)
	}
	if got := ratingSummary(report); got != "Missing risk rules for code impact, dependency changes in `gstyle.toml` at the base commit." {
		t.Fatalf("missing-rule categories: %s", got)
	}
	api := APIReport{Status: APIBreaking, Changes: []APIChange{{Package: "p", Message: "F removed"}}}
	report = Evaluate(Snapshot{}, Snapshot{}, config, api)
	if got := ratingSummary(report); got != "Missing risk rules for breaking API changes in `gstyle.toml` at the base commit." {
		t.Fatalf("missing API rule explanation: %s", got)
	}
}
