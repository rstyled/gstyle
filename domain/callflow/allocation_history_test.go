package callflow

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func TestAllocationHistoryAgainstFiniteSiteOracle(t *testing.T) {
	checked := 0
	for length := 0; length <= 5; length++ {
		limit := 1
		for range length {
			limit *= 3
		}
		for code := 0; code < limit; code++ {
			var history AllocationHistory
			seen := map[string]bool{}
			remaining := code
			for range length {
				site := fmt.Sprint(remaining % 3)
				remaining /= 3
				previous := history
				previousSites := previous.Sites()
				next, repeated := history.Extend(site)
				if repeated != seen[site] {
					t.Fatalf("history=%v site=%s repeated=%v", history.Sites(), site, repeated)
				}
				seen[site] = true
				expected := []string{}
				for s := range seen {
					expected = append(expected, s)
				}
				slices.Sort(expected)
				if !reflect.DeepEqual(next.Sites(), expected) {
					t.Fatalf("got %v want %v", next.Sites(), expected)
				}
				if !reflect.DeepEqual(previous.Sites(), previousSites) {
					t.Fatal("borrowed history changed")
				}
				history = next
			}
			checked++
		}
	}
	if checked != 364 {
		t.Fatalf("checked %d histories", checked)
	}
}

func TestAllocationHistoryOwnsOpaqueSites(t *testing.T) {
	sites := []string{"", "a\nb", "a\x00b", string([]byte{0xff}), string([]byte{0xfe}), "\ufffd"}
	var history AllocationHistory
	for _, site := range sites {
		next, repeated := history.Extend(site)
		if repeated {
			t.Fatalf("new opaque site repeated: %q", site)
		}
		history = next
	}
	expected := slices.Clone(sites)
	slices.Sort(expected)
	if !reflect.DeepEqual(history.Sites(), expected) {
		t.Fatalf("got %q want %q", history.Sites(), expected)
	}
	borrowed := history.Sites()
	borrowed[0] = "changed"
	if !reflect.DeepEqual(history.Sites(), expected) {
		t.Fatal("caller changed history")
	}
	for _, site := range sites {
		next, repeated := history.Extend(site)
		if !repeated || next != history {
			t.Fatalf("repeat changed history: %q", site)
		}
	}
}
