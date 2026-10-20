package goquality

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestAppendAssignmentsShareBudgetSite(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
 type State struct { Values []int }
 func Local() { xs:=[]int{}; xs=append(xs,1); xs=append(xs,2); _=xs }
 func Shared(xs []int) { xs=append(xs,1); xs=append(xs,2) }
 func Field(s *State) { s.Values=append(s.Values,1); s.Values=append(s.Values,2) }
 func Separate() { xs:=[]int{}; xs=append(xs,1); xs=append(xs,2); xs=append(xs,3); xs=append(xs,4); _=xs }
 func Other(xs []int) { var ys []int; ys=append(xs,1); _=ys }
 func Ordinary() { x:=0; x=1; x=2; x=3; x=4; _=x }
 func Parenthesized(s *State) { s.Values=(append)(s.Values,1); s.Values=append(s.Values,2) }
 func Nested(s *State) { s.Values=append(append(s.Values,1),2) }
 func Forced(xs []int) {var ys []int;ys=append(xs[:len(xs):len(xs)],1);_ =ys}
 `))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Saved facts retain operation identity and both storage effects.
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &facts); err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name           string
		events, budget int
		observable     bool
	}{
		{"Local", 4, 0, false}, {"Shared", 4, 0, true}, {"Field", 4, 0, true},
		{"Separate", 8, 1, false}, {"Other", 2, 0, true}, {"Ordinary", 4, 1, false},
		{"Parenthesized", 4, 0, true}, {"Nested", 3, 0, true},
		{"Forced", 2, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := findFunction(t, facts, test.name)
			if len(f.Mutations) != test.events {
				t.Fatalf("storage events: got %d want %d", len(f.Mutations), test.events)
			}
			count := 0
			for _, d := range report.Diagnostics {
				if d.Name == f.Name && d.RuleID == "mutation_per_target" {
					count++
				}
			}
			if count != test.budget {
				t.Fatalf("budget findings: got %d want %d", count, test.budget)
			}
			for _, result := range report.Functions {
				if result.Name == f.Name && (len(result.Effects) > 0) != test.observable {
					t.Fatalf("observable effects: %+v", result.Effects)
				}
				if result.Name != f.Name || (test.name != "Field" && test.name != "Parenthesized" && test.name != "Nested") {
					continue
				}
				sites := map[int]bool{}
				for _, effect := range result.Effects {
					if effect.Site == 0 {
						t.Fatalf("append effect lost operation identity: %+v", effect)
					}
					sites[effect.Site] = true
				}
				if len(sites) != 2 {
					t.Fatalf("distinct append operations collapsed: %+v", result.Effects)
				}
				if test.name != "Nested" && result.DensityMilli != 2000/int64(f.Statements) {
					t.Fatalf("append density counts storage evidence twice: %+v", result)
				}
			}
		})
	}
}
