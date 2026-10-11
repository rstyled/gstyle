package patterns

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// Check full and partial batches, remapped indices, and rejection by both gates.
func TestPairMatcherStripesPreserveAcceptedPairs(t *testing.T) {
	const size = 80
	ids := make([]string, size)
	indexes := make(map[string]int, size)
	candidates := newCCPairs(size + 3)
	expected := map[ccPairJob]int{}
	for i := range ids {
		ids[i] = fmt.Sprintf("f%d", i)
		indexes[ids[i]] = size + 2 - i
	}
	similar := func(a, b int) bool { return (a+b)%3 != 0 }
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			left, right := indexes[a], indexes[b]
			if (left+right)%2 != 0 {
				continue
			}
			candidates.add(right, left)
			if similar(left, right) {
				expected[ccPairJob{a: a, b: b}] = 1
			}
		}
	}
	matcher := newCCPairMatcher(ids, candidates, indexes, similar)
	for _, workers := range []int{1, 3, 8} {
		t.Run(fmt.Sprint(workers), func(t *testing.T) {
			found := make(chan []ccPairJob)
			var wg sync.WaitGroup
			for worker := range workers {
				wg.Add(1)
				go func() { defer wg.Done(); matcher.stripe(worker, workers, found) }()
			}
			go func() { wg.Wait(); close(found) }()
			got := map[ccPairJob]int{}
			for batch := range found {
				if len(batch) > 64 {
					t.Errorf("unbounded batch: %d", len(batch))
				}
				for _, pair := range batch {
					got[pair]++
				}
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("accepted pairs differ: got %d, want %d", len(got), len(expected))
			}
		})
	}
}
