package pdg

import (
	"slices"
	"testing"
)

func TestTablesTextBoundariesAndPreservation(t *testing.T) {
	table := Tables{Strings: []string{"first", "", "λ\x00last"}, Sources: &SourceContext{}}
	before := slices.Clone(table.Strings)
	source := table.Sources
	cases := []struct {
		id   Text
		want string
	}{
		{0, ""}, {1, "first"}, {2, ""}, {3, "λ\x00last"}, {4, ""}, {Text(^uint32(0)), ""},
	}
	for _, tc := range cases {
		if got := table.Text(tc.id); got != tc.want {
			t.Fatalf("id=%d got=%q want=%q", tc.id, got, tc.want)
		}
		if !slices.Equal(table.Strings, before) || table.Sources != source {
			t.Fatal("lookup mutates table")
		}
	}
}

func TestTablesTextEmptyStorage(t *testing.T) {
	for _, table := range []Tables{{}, {Strings: []string{}}} {
		for _, id := range []Text{0, 1, Text(^uint32(0))} {
			if got := table.Text(id); got != "" {
				t.Fatalf("id=%d got=%q", id, got)
			}
		}
	}
}

func TestTablesTextUsesLengthNotCapacity(t *testing.T) {
	backing := []string{"only", "hidden", "hidden2", "hidden3"}
	table := Tables{Strings: backing[:1]}
	if got := table.Text(1); got != "only" {
		t.Fatalf("first=%q", got)
	}
	for _, id := range []Text{2, 4} {
		if got := table.Text(id); got != "" {
			t.Fatalf("out-of-length id=%d got=%q", id, got)
		}
	}
	if len(table.Strings) != 1 || cap(table.Strings) != 4 {
		t.Fatal("lookup changes slice metadata")
	}
}
