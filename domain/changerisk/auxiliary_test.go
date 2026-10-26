package changerisk

import (
	"maps"
	"strings"
	"testing"
)

func TestAdditiveTestsDoNotHideExistingChanges(t *testing.T) {
	before := TestFile{Package: "sample", Imports: map[string]bool{"testing": true}, Declarations: map[string]string{"TestExisting": "before"}}
	for _, test := range []struct {
		name  string
		after map[string]TestFile
		want  Level
	}{
		{"new file", map[string]TestFile{"old_test.go": before, "new_test.go": before}, Low},
		{"unchanged", map[string]TestFile{"old_test.go": before}, Low},
		{"removed", nil, Unknown},
		{"changed test", map[string]TestFile{"old_test.go": {Package: "sample", Imports: before.Imports, Declarations: map[string]string{"TestExisting": "after"}}}, Unknown},
		{"removed test", map[string]TestFile{"old_test.go": {Package: "sample", Imports: before.Imports}}, Unknown},
		{"changed package", map[string]TestFile{"old_test.go": {Package: "sample_test", Imports: before.Imports, Declarations: before.Declarations}}, Unknown},
		{"removed import", map[string]TestFile{"old_test.go": {Package: "sample", Declarations: before.Declarations}}, Unknown},
		{"added test", map[string]TestFile{"old_test.go": {Package: "sample", Imports: before.Imports, Declarations: map[string]string{"TestExisting": "before", "TestNew": "new"}}}, Low},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := Evaluate(Snapshot{TestFiles: map[string]TestFile{"old_test.go": before}}, Snapshot{TestFiles: test.after}, Config{PDGChange: Low}, APIReport{})
			if report.Risk != test.want {
				t.Fatalf("risk=%s want=%s: %+v", report.Risk, test.want, report.Reasons)
			}
		})
	}
}

func TestBaselineCoverageNeverWeakensTheGate(t *testing.T) {
	base := QualityBaseline{Version: 1, Functions: 100, Tolerance: 0.05, Rules: map[string]BaselineRule{"density": {Limit: 500, Excess: 10, Rate: 100}}}
	for _, kind := range []string{"unchanged", "lower rate", "lower tolerance", "higher rate", "higher tolerance", "limit", "removed rule", "added rule", "version", "removed", "added"} {
		t.Run(kind, func(t *testing.T) {
			before, after := base, base
			after.Rules = maps.Clone(base.Rules)
			left, right := &before, &after
			want := Unknown
			switch kind {
			case "unchanged":
				want = Low
			case "lower rate":
				after.Rules["density"] = BaselineRule{Limit: 500, Excess: 9, Rate: 90}
				want = Low
			case "lower tolerance":
				after.Tolerance = 0.01
				want = Low
			case "higher rate":
				after.Rules["density"] = BaselineRule{Limit: 500, Excess: 11, Rate: 110}
			case "higher tolerance":
				after.Tolerance = 0.1
			case "limit":
				after.Rules["density"] = BaselineRule{Limit: 600, Excess: 10, Rate: 100}
			case "removed rule":
				delete(after.Rules, "density")
			case "added rule":
				after.Rules["new"] = BaselineRule{}
			case "version":
				after.Version = 2
			case "removed":
				right = nil
			case "added":
				left = nil
			}
			report := Evaluate(Snapshot{QualityBaseline: left}, Snapshot{QualityBaseline: right}, Config{PDGChange: Low}, APIReport{})
			if report.Risk != want {
				t.Fatalf("risk=%s want=%s: %+v", report.Risk, want, report.Reasons)
			}
		})
	}
}

func TestUnknownMarkdownShowsBlockingReasonsOnce(t *testing.T) {
	reason := Reason{Rule: "analysis_unavailable", Level: Unknown, Path: "quality-baseline.json", Message: "baseline needs review"}
	report := Report{Risk: Unknown, Reasons: []Reason{reason, reason, {Rule: "pdg_matching_ambiguous", Level: Unknown, Message: "graph illustration omitted"}}}
	text := Markdown(report)
	if strings.Count(text, "baseline needs review") != 1 || strings.Contains(text, "graph illustration omitted") {
		t.Fatalf("blocking reason summary: %s", text)
	}
}
