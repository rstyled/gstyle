package outputcheck

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestDiscardsBoundaries(t *testing.T) {
	counts := []int{math.MinInt, -1, 0, 1, 2, 3, math.MaxInt}
	for bits := 0; bits < 16; bits++ {
		for _, values := range counts {
			for _, results := range counts {
				input := Decision{Formatted: bits&1 != 0, Memory: bits&2 != 0, Standalone: bits&4 != 0, ErrorBlank: bits&8 != 0, Values: values, Results: results}
				want := input.Formatted && !input.Memory && (input.Standalone || values == 1 && results == 2 && input.ErrorBlank)
				if got := Discards(input); got != want {
					t.Fatalf("Discards(%+v) = %v, want %v", input, got, want)
				}
			}
		}
	}
}

func TestSelectAllSequences(t *testing.T) {
	for length := 0; length <= 8; length++ {
		for bits := 0; bits < 1<<length; bits++ {
			for _, suppressed := range []bool{false, true} {
				checkSelection(t, length, bits, suppressed)
			}
		}
	}
}

func checkSelection(t *testing.T, length, bits int, suppressed bool) {
	t.Helper()
	var sites, want []Site
	for i := 0; i < length; i++ {
		site := Site{FuncID: fmt.Sprint(i), Line: i + 1, Callee: fmt.Sprint("callee", i), Checked: i%2 == 0, DiscardedOutput: bits&(1<<i) != 0}
		sites = append(sites, site)
		if site.DiscardedOutput && !suppressed {
			want = append(want, site)
		}
	}
	before := append([]Site(nil), sites...)
	got := Select(sites, suppressed)
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(sites, before) {
		t.Fatalf("Select(%+v, %v) = %+v, want %+v; input=%+v", before, suppressed, got, want, sites)
	}
	if len(got) > 0 {
		got[0].Line = -1
		if !reflect.DeepEqual(sites, before) {
			t.Fatal("output aliases input")
		}
	}
}
