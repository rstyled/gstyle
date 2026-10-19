package quality

import "testing"

func TestMutationBudgetSitesKeepIndependentWrites(t *testing.T) {
	for _, test := range []struct {
		name string
		site int
		want int
	}{
		{"paired operation", 7, 2}, {"old saved facts", 0, 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutations := []Mutation{
				{Root: "xs", RootID: "1", Line: 1, Site: test.site, Provenance: Local},
				{Root: "xs", RootID: "1", Line: 1, Site: test.site, Provenance: External},
				{Root: "ys", RootID: "2", Line: 1, Site: test.site, Provenance: Local},
				{Root: "ys", RootID: "2", Line: 1, Site: test.site, Provenance: External},
			}
			total := 0
			for _, group := range mutationTargets(mutations, "root") {
				total += len(group)
			}
			if total != test.want {
				t.Fatalf("budget writes: got %d want %d", total, test.want)
			}
			if len(mutationEffects(mutations)) != 2 {
				t.Fatal("budget grouping hid external writes")
			}
		})
	}
}
