package quality

import "testing"

func TestMutationDensityRetainsOperationIdentity(t *testing.T) {
	for _, test := range []struct {
		name          string
		first, second int
		kind          Provenance
		weight        int64
	}{
		{"same operation", 7, 7, External, 1},
		{"different operations", 7, 8, External, 2},
		{"legacy facts", 0, 0, External, 2},
		{"different kinds", 7, 7, Unknown, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := fact("caller", 1)
			f.Mutations = []Mutation{
				{Root: "buffer", RootID: "1", Line: 2, Site: test.first, Provenance: External},
				{Root: "buffer", RootID: "1", Line: 2, Site: test.second, Provenance: test.kind},
			}
			got := evaluate(t, f, DefaultConfig()).Functions[0]
			want := 1000 * test.weight / int64(f.Statements)
			if got.DensityMilli != want || len(got.Effects) != 2 {
				t.Fatalf("density or raw evidence changed: %+v", got)
			}
			if got.Effects[0].Site != test.first || got.Effects[1].Site != test.second {
				t.Fatalf("operation identities lost: %+v", got.Effects)
			}
		})
	}
}

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
