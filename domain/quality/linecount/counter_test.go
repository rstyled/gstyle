package linecount

import (
	"math"
	"testing"
)

func TestCounterOrderedSpans(t *testing.T) {
	var counter Counter
	for _, span := range [][2]int{{1, 1}, {1, 1}, {3, 5}, {5, 5}, {5, 7}, {10, 10}} {
		next, ok := counter.Add(span[0], span[1])
		if !ok {
			t.Fatalf("rejected %v after %+v", span, counter)
		}
		counter = next
	}
	if counter != (Counter{Lines: 7, Through: 10}) {
		t.Fatalf("got %+v", counter)
	}
}

func TestCounterRejectsInvalidInputs(t *testing.T) {
	for _, test := range []struct {
		counter     Counter
		first, last int
	}{
		{Counter{Lines: -1}, 1, 1}, {Counter{Lines: 2, Through: 1}, 2, 2},
		{Counter{}, 0, 1}, {Counter{}, 2, 1}, {Counter{Lines: 1, Through: 3}, 2, 4},
	} {
		got, ok := test.counter.Add(test.first, test.last)
		if ok || got != test.counter {
			t.Fatalf("invalid input changes counter: %+v -> %+v, %v", test, got, ok)
		}
	}
}

func TestCounterIntegerBoundary(t *testing.T) {
	for _, initial := range []Counter{{}, {Lines: math.MaxInt - 1, Through: math.MaxInt - 1}, {Lines: math.MaxInt, Through: math.MaxInt}} {
		first := max(initial.Through, 1)
		got, ok := initial.Add(first, math.MaxInt)
		if !ok || got != (Counter{Lines: math.MaxInt, Through: math.MaxInt}) {
			t.Fatalf("boundary: %+v -> %+v, %v", initial, got, ok)
		}
	}
}

func TestExceedsStrictBoundary(t *testing.T) {
	for _, limit := range []int64{0, 200, math.MaxInt64 - 1} {
		if Exceeds(limit, limit) || !Exceeds(limit+1, limit) {
			t.Fatalf("threshold %d", limit)
		}
	}
}
