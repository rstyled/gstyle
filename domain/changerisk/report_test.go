package changerisk

import (
	"fmt"
	"strings"
	"testing"
)

func TestConfiguredCallsAndUnknownsRemainDistinct(t *testing.T) {
	before := Snapshot{Commit: "base", Functions: []Function{{Name: "p.Save", Path: "store.go", Line: 2, Body: "old", Graph: Graph{Shape: "old"}}}}
	after := Snapshot{Commit: "head", Functions: []Function{{Name: "p.Save", Path: "store.go", Line: 2, Body: "new", Graph: Graph{Shape: "new", Nodes: []Node{{Key: "write", Callee: "os.WriteFile", Label: "os.WriteFile", Line: 8}}}}}}
	config := Config{PDGChange: Medium, Calls: []CallRule{{Callee: "os.WriteFile", Level: High}}}
	report := Evaluate(before, after, config, APIReport{})
	if report.Risk != High || report.PDGRisk != High || len(report.Reasons) != 2 {
		t.Fatalf("configured risk lost: %+v", report)
	}
	if got := Evaluate(before, after, Config{}, APIReport{}); got.Risk != Unrated || got.PDGRisk != Unrated {
		t.Fatalf("unconfigured changes were rated: %+v", got)
	}
	markdown := Markdown(report)
	if strings.Count(strings.Split(markdown, "<details>")[0], "\n") > 20 || !strings.Contains(markdown, "**Merge risk: high** ⚠️") {
		t.Fatalf("summary is not compact: %s", markdown)
	}
}

func TestUnrepresentedSourceAndAmbiguousGraphsStayUnknown(t *testing.T) {
	before := Snapshot{Functions: []Function{{Name: "p.F", Body: "before", Graph: Graph{Shape: "same"}}}}
	after := Snapshot{Functions: []Function{{Name: "p.F", Body: "after", Graph: Graph{Shape: "same", Ambiguous: true}}}}
	report := Evaluate(before, after, Config{PDGChange: Low}, APIReport{})
	if report.PDGRisk != Unknown || report.Risk != Unknown || strings.Contains(Markdown(report), "```mermaid") || !strings.Contains(Markdown(report), "**Code impact:** partial") {
		t.Fatalf("incomplete comparison was treated as proof: %+v", report)
	}
}

func TestRepeatedCalleeCountDoesNotInventAnExactSite(t *testing.T) {
	before := Snapshot{Functions: []Function{{Name: "p.F", Line: 1, Body: "before", Graph: Graph{Shape: "before", Nodes: []Node{{Callee: "os.WriteFile", Line: 5}}}}}}
	after := Snapshot{Functions: []Function{{Name: "p.F", Line: 1, Body: "after", Graph: Graph{Shape: "after", Nodes: []Node{{Callee: "os.WriteFile", Line: 5}, {Callee: "os.WriteFile", Line: 9}}}}}}
	report := Evaluate(before, after, Config{Calls: []CallRule{{Callee: "os.WriteFile", Level: High}}}, APIReport{})
	if got := report.Reasons[len(report.Reasons)-1]; got.Rule != "added_call" || got.Line != 1 {
		t.Fatalf("ambiguous occurrence claimed an exact line: %+v", got)
	}
}

func TestMermaidEscapesSourceAndBoundsGraphs(t *testing.T) {
	delta := Delta{Name: "p.F", Path: "source.go"}
	for i := 0; i < 50; i++ {
		delta.Added = append(delta.Added, Node{Key: fmt.Sprint(i), Label: "\"\n```\n</details><script>alert(1)</script>"})
	}
	markdown := graphSection([]Delta{delta})
	if strings.Count(markdown, "[\"") != maxDiagramNodes || strings.Count(markdown, "```mermaid") != 1 || strings.Contains(markdown, "<script>") || strings.Contains(markdown, "\n```\n</details><script>") {
		t.Fatalf("unsafe or unbounded diagram: %s", markdown)
	}
	if got := graphSection(nil); got != "" {
		t.Fatalf("empty graph section: %s", got)
	}
}

func TestRiskConfigRejectsInvalidRules(t *testing.T) {
	for _, config := range []Config{
		{PDGChange: "typo"},
		{PDGLowRiskMaxFunctions: -1},
		{PDGLowRiskMaxFunctions: 3},
		{DependencyChange: Unknown},
		{PDGChange: Unrated},
		{Calls: []CallRule{{Callee: "", Level: High}}},
		{Calls: []CallRule{{Callee: "os.WriteFile"}}},
		{Calls: []CallRule{{Callee: "os.WriteFile", Level: High}, {Callee: "os.WriteFile", Level: Low}}},
	} {
		if err := config.Validate(); err == nil {
			t.Errorf("invalid policy accepted: %+v", config)
		}
	}
	if err := (Config{}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStaticRatingDoesNotInventUncollectedChecks(t *testing.T) {
	before := Snapshot{Functions: []Function{{Name: "p.F", Body: "old", Graph: Graph{Shape: "old"}}}}
	after := Snapshot{Functions: []Function{{Name: "p.F", Body: "new", Graph: Graph{Shape: "new", Ambiguous: true}}}}
	report := Evaluate(before, after, Config{PDGChange: Medium}, APIReport{})
	if report.Risk != Medium || report.PDGRisk != Medium {
		t.Fatalf("uncollected checks or diagram limits replaced the measured static rating: %+v", report)
	}
	markdown := Markdown(report)
	for _, placeholder := range []string{"**Compatibility:**", "**Checks:**", "**Coverage:**", "**Analysis limits:**", "diagrams omitted", "Analysis details", "Not assessed:", "Compared "} {
		if strings.Contains(markdown, placeholder) {
			t.Fatalf("unimplemented category shown as a verdict: %s", markdown)
		}
	}
	for _, evidence := range []string{"**Code impact:** changed 👀  \nAffected functions: 1."} {
		if !strings.Contains(markdown, evidence) {
			t.Errorf("missing %q in %s", evidence, markdown)
		}
	}
	if strings.Join(report.Unmeasured, ",") != "go_api,cli,checks,test_coverage" {
		t.Fatalf("uncollected scopes lost from structured evidence: %+v", report.Unmeasured)
	}
}

func TestDependencyFactsDoNotDependOnSeverityPolicy(t *testing.T) {
	base := Snapshot{Dependencies: []string{".:import:pkg -> old", ".:module:old"}}
	head := Snapshot{Dependencies: []string{".:import:pkg -> new", ".:module:new"}}
	report := Evaluate(base, head, Config{}, APIReport{})
	if report.Risk != Unrated || report.DependencyRisk != Unrated || !report.DependenciesChanged {
		t.Fatalf("missing policy hid measured changes: %+v", report)
	}
	if len(report.DependenciesAdded) != 2 || len(report.DependenciesRemoved) != 2 {
		t.Fatalf("dependency deltas missing: %+v", report)
	}
	markdown := Markdown(report)
	if !strings.Contains(markdown, "**Dependencies:** changed") || !strings.Contains(markdown, "Selected module versions unchanged.") {
		t.Fatalf("dependency facts missing: %s", markdown)
	}
	head.DependencyError = "enumeration unavailable"
	report = Evaluate(base, head, Config{DependencyChange: High}, APIReport{})
	if len(report.DependenciesAdded)+len(report.DependenciesRemoved) != 0 || report.Risk != Unknown || !strings.Contains(Markdown(report), "**Dependencies:** unavailable") {
		t.Fatalf("failed enumeration invented a delta or verdict: %+v", report)
	}
}

func TestUnresolvedTargetsRemainVisibleWithUnratedPolicy(t *testing.T) {
	head := Snapshot{Functions: []Function{{Name: "p.F", Body: "new", Graph: Graph{Shape: "new", Nodes: []Node{{Key: "call", UnresolvedCall: true}}}}}}
	report := Evaluate(Snapshot{}, head, Config{}, APIReport{})
	if report.Risk != Unrated || report.PDGRisk != Unknown {
		t.Fatalf("missing policy and analysis gap were conflated: %+v", report)
	}
	markdown := Markdown(report)
	if !strings.Contains(markdown, "**Code impact:** changed") || unknownReasons(report.Reasons, "unresolved_calls") != 1 {
		t.Fatalf("real analysis gap lost from structured evidence: %+v", report)
	}
	if strings.Contains(markdown, "unresolved call targets") {
		t.Fatalf("removed analysis counters still appear in Markdown: %s", markdown)
	}
}

func TestUnavailableEvidenceAndPriorityRemainVisible(t *testing.T) {
	report := Evaluate(Snapshot{DependencyError: "not available"}, Snapshot{}, Config{DependencyChange: Low}, APIReport{})
	if report.DependencyRisk != Unknown || report.Risk != Unknown || !strings.Contains(Markdown(report), "not available") {
		t.Fatalf("missing dependency evidence became a pass: %+v", report)
	}
	report.Changes = []Delta{{Name: "a", Added: []Node{{Key: "a", Label: "a"}}}, {Name: "z", Added: []Node{{Key: "z", Label: "z"}}}}
	report.Reasons = []Reason{{Function: "a", Level: Low}, {Function: "z", Level: High}}
	if first := prioritizedChanges(report)[0].Name; first != "z" {
		t.Fatalf("high-risk graph was not prioritized: %s", first)
	}
}
