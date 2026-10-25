package changerisk

import (
	"strings"
	"testing"
)

func TestDependencySummaryPairsVersionsWithoutCountingGraphEdgesAsLibraries(t *testing.T) {
	report := Report{DependenciesChanged: true,
		DependenciesRemoved: []string{`.:module:"example.com/lib":"v1.0.0":"1.25"`},
		DependenciesAdded: []string{`.:module:"example.com/lib":"v1.1.0":"1.26"`, `.:module:"example.com/new":"v2.0.0":"1.26"`,
			`.:example.com/main example.com/lib@v1.1.0`, `.:example.com/lib@v1.1.0 go@1.26`, `.:import:example.com/main -> example.com/lib`}}
	markdown := Markdown(report)
	for _, want := range []string{"Imports: **+1 / −0**.", "Updated `example.com/lib`: `v1.0.0` → `v1.1.0`.", "Added `example.com/new` at `v2.0.0`."} {
		if !strings.Contains(markdown, want) {
			t.Errorf("missing %q in %s", want, markdown)
		}
	}
	changes := moduleChanges(report)
	if len(changes) != 2 || changes[0].before != "v1.0.0" || changes[0].after != "v1.1.0" {
		t.Fatalf("module edges became libraries or update was split: %+v", changes)
	}
	report.DependenciesRemoved = append([]string(nil), report.DependenciesAdded[:1]...)
	report.DependenciesAdded[0] = `.:module:"example.com/lib":"v1.1.0":"1.27"`
	if got := moduleChangeLine(moduleChanges(report)[0]); !strings.Contains(got, "Updated module settings") {
		t.Fatalf("Go version metadata became an upgrade: %s", got)
	}
}

func TestLongChangeListsRemainAvailableBelowCompactSummary(t *testing.T) {
	report := Report{GoAPI: APIReport{Status: APICompatible}}
	for i := range 12 {
		report.GoAPI.Changes = append(report.GoAPI.Changes, APIChange{Package: "example.com/lib", Message: string(rune('A'+i)) + ": added", Compatible: true})
	}
	report.GoAPI.Changes = append(report.GoAPI.Changes, APIChange{Package: "example.com/lib", Message: "Removed: removed"})
	markdown := Markdown(report)
	summary := strings.Split(markdown, "<details>")[0]
	if strings.Count(summary, "\n") > 20 || !strings.Contains(summary, "lib.Removed") || strings.Count(markdown, "lib.L") != 1 {
		t.Fatalf("summary hid a break, exceeded bounds, or lost full detail: %s", markdown)
	}
}
