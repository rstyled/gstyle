package application

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/domain/quality"
)

func TestUnknownOnlyDensityDoesNotCreateCountsOrNavigationStops(t *testing.T) {
	m := qualityModel()
	q := m.report.Files[0].Functions[0].Quality
	q.Diagnostics = []quality.Diagnostic{{
		Location: q.Location, RuleID: "side_effect_density", Actual: 2166, Limit: 500,
		Weight: 26, Statements: 12, KindWeights: map[string]int64{"unknown": 25, "global_read": 1},
		Effects: []quality.Effect{{Line: 12, Kind: quality.UnknownEffect, Detail: "unresolved target"}, {Line: 13, Kind: quality.GlobalRead, Detail: "locale read"}},
	}}
	q.Effects = q.Diagnostics[0].Effects
	m = m.applyReport(m.report)
	raw, err := json.Marshal(m.controlReport())
	if err != nil || !strings.Contains(string(raw), "unresolved target") {
		t.Fatal("hiding unknown effects changes the raw JSON evidence")
	}
	plain := ansi.Strip(strings.Join(m.sourceLines(130, 100, "Source"), "\n"))
	if len(m.findingTargets()) != 0 || ansi.Strip(qualityBadge(m.report.Files[0].Functions[0])) != " · Q 0" || m.qualityHeader() != " · Q 0" {
		t.Fatal("unknown-only density affects badges or bracket navigation")
	}
	if strings.Contains(plain, "quality:") || strings.Contains(plain, "unresolved") || strings.Contains(strings.Join(m.qualityBody(), "\n"), "2166") {
		t.Fatalf("an estimate is still presented as a finding:\n%s", plain)
	}
	if !strings.Contains(strings.Join(m.qualityBody(), "\n"), "Known density 83") || len(q.Effects) != 2 || q.Diagnostics[0].Actual != 2166 {
		t.Fatal("the visible density or original evidence changes incorrectly")
	}
}

func TestKnownDensityExcessSurvivesUnknownContributions(t *testing.T) {
	m := qualityModel()
	q := m.report.Files[0].Functions[0].Quality
	q.Diagnostics = []quality.Diagnostic{{
		Location: q.Location, RuleID: "side_effect_density", Actual: 3100, Limit: 500,
		Weight: 31, Statements: 10, KindWeights: map[string]int64{"unknown": 25, "io": 6},
		Effects: []quality.Effect{{Site: 1, Line: 12, Kind: quality.IO, Detail: "write"}, {Site: 2, Line: 13, Kind: quality.UnknownEffect, Detail: "unresolved"}},
	}}
	q.Effects = q.Diagnostics[0].Effects
	before, _ := json.Marshal(q)
	visible := knownQuality(q)
	if len(visible.Diagnostics) != 1 || visible.Diagnostics[0].Actual != 600 || len(visible.Diagnostics[0].Effects) != 1 || visible.Diagnostics[0].KindWeights["unknown"] != 0 {
		t.Fatalf("known effects above the limit disappear or retain unknown weight: %+v", visible.Diagnostics)
	}
	view := strings.Join(functionQualityLines(q), "\n")
	if strings.Contains(view, "unresolved") || !strings.Contains(view, "L12 io: write") {
		t.Fatal("the quality pane hides known effects or displays unknown effects")
	}
	after, _ := json.Marshal(q)
	if string(before) != string(after) {
		t.Fatal("filtering changes raw diagnostics, effects, or weights")
	}
}

func TestUnknownMutationBudgetsRequireKnownExcess(t *testing.T) {
	for _, knownCount := range []int{3, 4} {
		for _, rule := range []string{"mutated_targets", "mutation_per_target"} {
			m := qualityModel()
			q := m.report.Files[0].Functions[0].Quality
			q.MutationEvents = nil
			for index := range 5 {
				provenance := quality.Unknown
				if index < knownCount {
					provenance = quality.External
				}
				root := "state"
				if rule == "mutated_targets" {
					root = fmt.Sprintf("state%d", index)
				}
				q.MutationEvents = append(q.MutationEvents, quality.Mutation{Root: root, RootID: root, Provenance: provenance, Line: 11})
			}
			q.Mutations, q.MutatedTargets = 5, 5
			q.Diagnostics = []quality.Diagnostic{{Location: q.Location, RuleID: rule, Actual: 5, Limit: 3}}
			if rule == "mutation_per_target" {
				q.Diagnostics[0].Mutations = q.MutationEvents
			}
			visible := knownQuality(q)
			if len(visible.Diagnostics) != knownCount-3 {
				t.Fatalf("%s with %d known writes: %+v", rule, knownCount, visible.Diagnostics)
			}
			if len(visible.Diagnostics) == 1 && visible.Diagnostics[0].Actual != 4 {
				t.Fatal("a known mutation finding keeps the unknown contribution")
			}
		}
	}
}

func TestKnownTargetProjectionPreservesFieldGranularity(t *testing.T) {
	var all []quality.Mutation
	for index := range 5 {
		provenance := quality.External
		if index == 4 {
			provenance = quality.Unknown
		}
		all = append(all, quality.Mutation{Root: "state", RootID: "p", FieldPath: fmt.Sprint(index), Provenance: provenance})
	}
	count, supported := knownTargetCount(5, all, knownMutations(all))
	if !supported || count != 4 {
		t.Fatal("field-level known targets are counted as one root")
	}
}

func TestOpaqueLegacyDensityStaysOutOfTheTUI(t *testing.T) {
	diagnostic := quality.Diagnostic{RuleID: "side_effect_density", Actual: 2000, Limit: 500, Effects: []quality.Effect{{Kind: quality.UnknownEffect}}}
	if _, visible := knownDensityDiagnostic(diagnostic); visible {
		t.Fatal("missing weight metadata makes an unknown estimate visible")
	}
}
