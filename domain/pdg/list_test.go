package pdg

import (
	"slices"
	"testing"
)

func TestListAtReferenceRows(t *testing.T) {
	checkListAtRows(t, [][]Ref{nil, {}, make([]Ref, 0, 2), {7, 9}})
}

func TestListAtAccessRows(t *testing.T) {
	checkListAtRows(t, [][]Access{nil, {}, {{Symbol: 3, Position: 1}}})
}

func TestListAtExtensionRows(t *testing.T) {
	checkListAtRows(t, [][]Extension{nil, {}, {{Name: 2, Value: 4}}})
}

func checkListAtRows[T comparable](t *testing.T, rows [][]T) {
	t.Helper()
	before := slices.Clone(rows)
	contents := make([][]T, len(rows))
	for i := range rows {
		contents[i] = slices.Clone(rows[i])
	}
	for i, want := range rows {
		got := listAt(rows, Ref(i+1))
		if !slices.Equal(got, want) || (got == nil) != (want == nil) || cap(got) != cap(want) {
			t.Fatalf("ref=%d got=%v want=%v; slice identity differs", i+1, got, want)
		}
		if cap(want) > 0 && &got[:cap(got)][0] != &want[:cap(want)][0] {
			t.Fatalf("ref=%d copies the row", i+1)
		}
	}
	for _, ref := range []Ref{0, Ref(len(rows) + 1), Ref(^uint32(0))} {
		if got := listAt(rows, ref); got != nil {
			t.Fatalf("ref=%d got=%v want nil", ref, got)
		}
	}
	for i := range rows {
		if !slices.Equal(rows[i], contents[i]) || (rows[i] == nil) != (before[i] == nil) || cap(rows[i]) != cap(before[i]) {
			t.Fatal("lookup changes table descriptors")
		}
		if cap(rows[i]) > 0 && &rows[i][:cap(rows[i])][0] != &before[i][:cap(before[i])][0] {
			t.Fatal("lookup changes a row pointer")
		}
	}
}

func TestListAtOuterSliceBoundaries(t *testing.T) {
	for _, rows := range [][][]Ref{nil, {}, make([][]Ref, 0, 2)} {
		for _, ref := range []Ref{0, 1, Ref(^uint32(0))} {
			if got := listAt(rows, ref); got != nil {
				t.Fatalf("empty table ref=%d got=%v", ref, got)
			}
		}
	}
	rows := [][]Ref{{4}, {8}, {12}}
	if got := listAt(rows[:1], 2); got != nil {
		t.Fatalf("spare capacity becomes a present row: %v", got)
	}
}
