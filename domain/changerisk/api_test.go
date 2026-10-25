package changerisk

import (
	"strings"
	"testing"
)

func TestAPISeverityAndUnmeasuredScopesStaySeparate(t *testing.T) {
	api := APIReport{Status: APIBreaking, PackagesBefore: 1, PackagesAfter: 1, Changes: []APIChange{{Package: "example.com/lib", Message: "Save: signature changed"}}}
	report := Evaluate(Snapshot{}, Snapshot{}, Config{APIBreaking: High}, api)
	if report.Risk != High || report.APIRisk != High || report.PDGRisk != Low {
		t.Fatalf("API break did not receive its own configured severity: %+v", report)
	}
	markdown := Markdown(report)
	for _, want := range []string{"**API diff:** breaking changes", "`lib.Save`: signature changed", "Not assessed: CLI contract, check results, runtime coverage."} {
		if !strings.Contains(markdown, want) {
			t.Errorf("missing %q in %s", want, markdown)
		}
	}
	if strings.Count(strings.Split(markdown, "<details>")[0], "\n") > 20 {
		t.Fatalf("API evidence exceeded the compact summary: %s", markdown)
	}
	unrated := Evaluate(Snapshot{}, Snapshot{}, Config{}, api)
	if unrated.Risk != Unrated || unrated.GoAPI.Status != APIBreaking {
		t.Fatalf("missing policy hid a known API break: %+v", unrated)
	}
	if err := (Config{APIBreaking: Unknown}).Validate(); err == nil {
		t.Fatal("invalid API risk policy accepted")
	}
}

func TestUnavailableAndPartialAPIAreNotCompatibilityPasses(t *testing.T) {
	for _, api := range []APIReport{
		{Status: APIUnavailable, Error: "type loading failed"},
		{Status: APIPartial, Limits: []string{"generic constraints changed"}},
	} {
		report := Evaluate(Snapshot{}, Snapshot{}, Config{APIBreaking: High}, api)
		if report.Risk != Unknown || report.APIRisk != Unknown {
			t.Fatalf("missing API evidence became a pass: %+v", report)
		}
		if strings.Contains(Markdown(report), "**API diff:** unchanged") {
			t.Fatalf("missing API evidence rendered as unchanged: %+v", report)
		}
	}
}
