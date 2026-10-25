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
	report := Evaluate(before, after, config)
	if report.Risk != High || report.PDGRisk != High || len(report.Reasons) != 2 {
		t.Fatalf("configured risk lost: %+v", report)
	}
	if got := Evaluate(before, after, Config{}); got.Risk != Unknown || got.PDGRisk != Unknown {
		t.Fatalf("unconfigured changes were rated: %+v", got)
	}
	markdown := Markdown(report)
	if strings.Count(strings.Split(markdown, "<details>")[0], "\n") > 20 || !strings.Contains(markdown, "**Change risk: high**") {
		t.Fatalf("summary is not compact: %s", markdown)
	}
}

func TestUnrepresentedSourceAndAmbiguousGraphsStayUnknown(t *testing.T) {
	before := Snapshot{Functions: []Function{{Name: "p.F", Body: "before", Graph: Graph{Shape: "same"}}}}
	after := Snapshot{Functions: []Function{{Name: "p.F", Body: "after", Graph: Graph{Shape: "same", Ambiguous: true}}}}
	report := Evaluate(before, after, Config{PDGChange: Low})
	if report.PDGRisk != Unknown || report.Risk != Unknown || strings.Contains(Markdown(report), "```mermaid") {
		t.Fatalf("incomplete comparison was treated as proof: %+v", report)
	}
}

func TestRepeatedCalleeCountDoesNotInventAnExactSite(t *testing.T) {
	before := Snapshot{Functions: []Function{{Name: "p.F", Line: 1, Body: "before", Graph: Graph{Shape: "before", Nodes: []Node{{Callee: "os.WriteFile", Line: 5}}}}}}
	after := Snapshot{Functions: []Function{{Name: "p.F", Line: 1, Body: "after", Graph: Graph{Shape: "after", Nodes: []Node{{Callee: "os.WriteFile", Line: 5}, {Callee: "os.WriteFile", Line: 9}}}}}}
	report := Evaluate(before, after, Config{Calls: []CallRule{{Callee: "os.WriteFile", Level: High}}})
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
		{DependencyChange: Unknown},
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

func TestUnavailableEvidenceAndPriorityRemainVisible(t *testing.T) {
	report := Evaluate(Snapshot{DependencyError: "not available"}, Snapshot{}, Config{DependencyChange: Low})
	if report.DependencyRisk != Unknown || report.Risk != Unknown || !strings.Contains(Markdown(report), "not available") {
		t.Fatalf("missing dependency evidence became a pass: %+v", report)
	}
	report.Changes = []Delta{{Name: "a", Added: []Node{{Key: "a", Label: "a"}}}, {Name: "z", Added: []Node{{Key: "z", Label: "z"}}}}
	report.Reasons = []Reason{{Function: "a", Level: Low}, {Function: "z", Level: High}}
	if first := prioritizedChanges(report)[0].Name; first != "z" {
		t.Fatalf("high-risk graph was not prioritized: %s", first)
	}
}
