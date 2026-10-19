package quality

import (
	"reflect"
	"testing"
)

func TestCallDensityCountsDistinctSitesOnOneLine(t *testing.T) {
	f := fact("project.caller", 1)
	f.Calls = []Call{
		{Callee: "project.helper", Local: true, Line: 2, Site: 10},
		{Callee: "project.helper", Local: true, Line: 2, Site: 20},
	}
	f.Helpers = []Helper{{Name: "project.helper", Effects: []Effect{
		{Kind: IO, Detail: "first write", Line: 10},
		{Kind: IO, Detail: "second write", Line: 11},
	}}}
	got := evaluate(t, f, DefaultConfig()).Functions[0]
	if got.DensityMilli != 1500 || len(got.Effects) != 4 || !got.Complete {
		t.Fatalf("distinct calls on one line: %+v", got)
	}
	counts := map[int]int{}
	for _, effect := range got.Effects {
		counts[effect.Site]++
		if effect.Line != 2 {
			t.Fatalf("caller evidence line: %+v", effect)
		}
	}
	if !reflect.DeepEqual(counts, map[int]int{10: 2, 20: 2}) {
		t.Fatalf("call identities lost: %+v", got.Effects)
	}
}

func TestCallDensityRetainsEachKindAndRawDetails(t *testing.T) {
	f := fact("project.caller", 1)
	f.Calls = []Call{{Callee: "project.helper", Local: true, Line: 2, Site: 10}}
	f.Helpers = []Helper{{Name: "project.helper", Effects: []Effect{
		{Kind: IO, Detail: "first write", Line: 10},
		{Kind: IO, Detail: "second write", Line: 11},
		{Kind: Global, Detail: "first global", Line: 12},
		{Kind: Global, Detail: "second global", Line: 13},
		{Kind: UnknownEffect, Detail: "unproved intrinsic", Line: 14},
	}}}
	report := evaluate(t, f, DefaultConfig())
	got := report.Functions[0]
	if got.DensityMilli != 1500 || len(got.Effects) != 5 || got.Complete {
		t.Fatalf("kind accounting: %+v", got)
	}
	details := map[string]bool{}
	for _, effect := range got.Effects {
		details[effect.Detail] = true
		if effect.Site != 10 {
			t.Fatalf("raw evidence has wrong caller identity: %+v", effect)
		}
	}
	if len(details) != 5 {
		t.Fatalf("raw details collapsed: %+v", got.Effects)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.RuleID != "side_effect_density" {
			continue
		}
		if diagnostic.Weight != 6 || !reflect.DeepEqual(diagnostic.KindWeights, map[string]int64{"io": 3, "global": 2, "unknown": 1}) || len(diagnostic.Effects) != 5 {
			t.Fatalf("diagnostic arithmetic differs from function: %+v", diagnostic)
		}
		return
	}
	t.Fatal("missing density diagnostic")
}

func TestCallDensityUsesOutermostCallAndChecksHelperSeparately(t *testing.T) {
	leaf := Helper{Name: "project.leaf", Effects: []Effect{
		{Kind: IO, Detail: "first write", Line: 20, Site: 100},
		{Kind: IO, Detail: "second write", Line: 21, Site: 110},
	}}
	inner := fact("project.inner", 10)
	inner.Calls = []Call{
		{Callee: "project.leaf", Local: true, Line: 11, Site: 30},
		{Callee: "project.leaf", Local: true, Line: 12, Site: 40},
	}
	inner.Helpers = []Helper{leaf}
	outer := fact("project.outer", 1)
	outer.Calls = []Call{{Callee: inner.Name, Local: true, Line: 2, Site: 10}}
	outer.Helpers = []Helper{{Name: inner.Name, Calls: inner.Calls}, leaf}
	report, err := Evaluate([]Function{outer, inner}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range report.Functions {
		want := int64(750)
		if got.Name == inner.Name {
			want = 1500
		}
		if got.DensityMilli != want || len(got.Effects) != 4 {
			t.Fatalf("nested density: %+v", got)
		}
		if got.Name == outer.Name {
			for _, effect := range got.Effects {
				if effect.Site != 10 || effect.Line != 2 {
					t.Fatalf("inner site leaked into outer density: %+v", effect)
				}
			}
		}
	}
}

func TestCallDensityLegacyFactsKeepIndependentEffects(t *testing.T) {
	f := fact("project.caller", 1)
	f.Calls = []Call{{Callee: "project.helper", Local: true, Line: 2}}
	f.Helpers = []Helper{{Name: "project.helper", Effects: []Effect{
		{Kind: IO, Detail: "first write", Line: 10, Site: 10},
		{Kind: IO, Detail: "second write", Line: 11, Site: 10},
	}}}
	got := evaluate(t, f, DefaultConfig()).Functions[0]
	if got.DensityMilli != 1500 || len(got.Effects) != 2 {
		t.Fatalf("legacy outer call silently deduplicates: %+v", got)
	}
	for _, effect := range got.Effects {
		if effect.Site != 0 {
			t.Fatalf("legacy outer call retains inner identity: %+v", effect)
		}
	}
}

func TestCallDensityKeepsUnknownCoverageAndRawCallCount(t *testing.T) {
	f := fact("project.caller", 1)
	f.Calls = []Call{{Callee: "project.helper", Local: true, Line: 2, Site: 10}}
	f.Helpers = []Helper{{Name: "project.helper", Calls: []Call{
		{Callee: "missing.one", Line: 10, Site: 20},
		{Callee: "missing.two", Line: 11, Site: 30},
	}}}
	got := evaluate(t, f, DefaultConfig()).Functions[0]
	if got.DensityMilli != 250 || got.Complete || got.UnclassifiedCalls != 2 || len(got.Effects) != 2 {
		t.Fatalf("density change hides uncertainty: %+v", got)
	}
}

func TestCallDensityRejectsNegativeSites(t *testing.T) {
	f := fact("project.caller", 1)
	f.Calls = []Call{{Callee: "unknown", Line: 2, Site: -1}}
	if _, err := Evaluate([]Function{f}, DefaultConfig()); err == nil {
		t.Fatal("negative call site accepted")
	}
	f.Calls = nil
	f.Effects = []Effect{{Kind: IO, Line: 2, Site: -1}}
	if _, err := Evaluate([]Function{f}, DefaultConfig()); err == nil {
		t.Fatal("negative effect site accepted")
	}
}
