package changerisk

import (
	"fmt"
	"testing"
)

func TestSmallChangeThreshold(t *testing.T) {
	for _, tc := range []struct {
		name           string
		count, maximum int
		severity, want Level
	}{
		{"none", 0, 3, Medium, Low},
		{"one", 1, 3, Medium, Low},
		{"boundary", 3, 3, Medium, Low},
		{"over", 4, 3, Medium, Medium},
		{"disabled", 1, 0, Medium, Medium},
		{"high policy", 1, 3, High, High},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, head := changedSnapshots(tc.count)
			config := Config{PDGChange: tc.severity, PDGLowRiskMaxFunctions: tc.maximum}
			report := Evaluate(base, head, config, APIReport{})
			if report.Risk != tc.want || report.PDGRisk != tc.want || report.ChangedFunctions != tc.count {
				t.Fatalf("unexpected threshold rating: %+v", report)
			}
			for _, reason := range report.Reasons {
				if reason.Rule == "pdg_change" && reason.Level != tc.want {
					t.Fatalf("reason disagrees with rating: %+v", reason)
				}
			}
		})
	}
}

func TestSmallChangesPreserveOtherRiskEvidence(t *testing.T) {
	for _, kind := range []string{"call", "unresolved", "projection", "coverage", "declaration", "dependency", "api"} {
		t.Run(kind, func(t *testing.T) {
			base, head := changedSnapshots(1)
			config := Config{PDGChange: Medium, PDGLowRiskMaxFunctions: 3, DependencyChange: High, APIBreaking: High,
				Calls: []CallRule{{Callee: "os.WriteFile", Level: High}}}
			api := APIReport{}
			want := Unknown
			switch kind {
			case "call":
				head.Functions[0].Graph.Nodes = []Node{{Key: "write", Callee: "os.WriteFile"}}
				want = High
			case "unresolved":
				head.Functions[0].Graph.Nodes = []Node{{Key: "call", UnresolvedCall: true}}
			case "projection":
				head.Functions[0].Graph.Shape = base.Functions[0].Graph.Shape
			case "coverage":
				head.AnalysisError = "workflow change requires review"
			case "declaration":
				head.Declarations = map[string]string{"source.go": "new declaration"}
			case "dependency":
				head.Dependencies = []string{"new dependency"}
				want = High
			case "api":
				api = APIReport{Status: APIBreaking, Changes: []APIChange{{Package: "p", Message: "F removed"}}}
				want = High
			}
			if report := Evaluate(base, head, config, api); report.Risk != want {
				t.Fatalf("small change hid %s evidence: %+v", kind, report)
			}
		})
	}
}

func changedSnapshots(count int) (Snapshot, Snapshot) {
	base, head := Snapshot{}, Snapshot{}
	for index := 0; index < count; index++ {
		name := fmt.Sprintf("p.F%d", index)
		base.Functions = append(base.Functions, Function{Name: name, Body: "before", Graph: Graph{Shape: "before"}})
		head.Functions = append(head.Functions, Function{Name: name, Body: "after", Graph: Graph{Shape: "after"}})
	}
	return base, head
}
