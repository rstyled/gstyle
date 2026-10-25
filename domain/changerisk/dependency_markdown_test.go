package changerisk

import (
	"strings"
	"testing"
)

func TestDependencySummaryPairsVersionsWithoutCountingGraphEdgesAsLibraries(t *testing.T) {
	report := Report{DependenciesChanged: true,
		DependenciesRemoved: []string{`.:module:"example.com/lib":"v1.0.0":"1.25"`, `.:module:"example.com/old":"v1.0.0":"1.25"`},
		DependenciesAdded: []string{`.:module:"example.com/lib":"v1.1.0":"1.26"`, `.:module:"example.com/new":"v2.0.0":"1.26"`, `.:module:"example.com/extra":"v1.0.0":"1.26"`,
			`.:example.com/main example.com/lib@v1.1.0`, `.:example.com/lib@v1.1.0 go@1.26`, `.:import:example.com/main -> example.com/lib`}}
	markdown := Markdown(report)
	summary := strings.Split(markdown, "<details>")[0]
	for _, want := range []string{"Changes: 1 updated, 2 added, 1 removed.", "Updated `example.com/lib`: `v1.0.0` → `v1.1.0`.", "Added `example.com/new` at `v2.0.0`.", "Added `example.com/extra` at `v1.0.0`.", "Removed `example.com/old` at `v1.0.0`."} {
		if strings.Count(markdown, want) != 1 || !strings.Contains(summary, want) {
			t.Errorf("expected one visible occurrence of %q in %s", want, markdown)
		}
	}
	for _, unwanted := range []string{"Imports:", "Remaining module changes", "more module changes"} {
		if strings.Contains(markdown, unwanted) {
			t.Errorf("obsolete dependency output %q in %s", unwanted, markdown)
		}
	}
	changes := moduleChanges(report)
	if len(changes) != 4 || changes[0].before != "v1.0.0" || changes[0].after != "v1.1.0" {
		t.Fatalf("module edges became libraries or update was split: %+v", changes)
	}
	report.DependenciesRemoved = append([]string(nil), report.DependenciesAdded[:1]...)
	report.DependenciesAdded[0] = `.:module:"example.com/lib":"v1.1.0":"1.27"`
	if got := moduleChangeLine(moduleChanges(report)[0]); !strings.Contains(got, "Updated module settings") {
		t.Fatalf("Go version metadata became an upgrade: %s", got)
	}
}

func TestImportAndRequirementOnlyChangesRemainVisibleWithoutModuleCounts(t *testing.T) {
	for _, records := range [][2]string{
		{".:import:example.com/main -> old", ".:import:example.com/main -> new"},
		{".:example.com/main example.com/lib@v1.0.0", ".:example.com/main example.com/lib@v1.1.0"},
	} {
		report := Evaluate(Snapshot{Dependencies: []string{records[0]}}, Snapshot{Dependencies: []string{records[1]}}, Config{DependencyChange: High}, APIReport{})
		markdown := Markdown(report)
		if report.DependencyRisk != High || !report.DependenciesChanged || len(moduleChanges(report)) != 0 || len(report.DependenciesAdded) != 1 || len(report.DependenciesRemoved) != 1 {
			t.Fatalf("non-module dependency evidence changed: %+v", report)
		}
		if !strings.Contains(markdown, "**Dependencies:** changed") || !strings.Contains(markdown, "Package imports or module requirements changed.") || strings.Contains(markdown, "Imports:") {
			t.Fatalf("non-module changes were hidden or counted as modules: %s", markdown)
		}
	}
}

func TestAPIChangesRenderOnceWithoutAnalysisDetails(t *testing.T) {
	report := Report{GoAPI: APIReport{Status: APICompatible}}
	for i := range 12 {
		report.GoAPI.Changes = append(report.GoAPI.Changes, APIChange{Package: "example.com/lib", Message: string(rune('A'+i)) + ": added", Compatible: true})
	}
	report.GoAPI.Changes = append(report.GoAPI.Changes, APIChange{Package: "example.com/lib", Message: "Removed: removed"})
	markdown := Markdown(report)
	summary := strings.Split(markdown, "<details>")[0]
	if strings.Count(markdown, "**API diff:**") != 1 || strings.Contains(markdown, "Analysis details") || !strings.Contains(summary, "lib.Removed") {
		t.Fatalf("API list is duplicated, hidden, or missing its break: %s", markdown)
	}
	for _, change := range report.GoAPI.Changes {
		name, _, _ := strings.Cut(change.Message, ":")
		if strings.Count(markdown, "`lib."+name+"`") != 1 || !strings.Contains(summary, "`lib."+name+"`") {
			t.Errorf("API change %s is hidden, duplicated, or missing: %s", name, markdown)
		}
	}
}
