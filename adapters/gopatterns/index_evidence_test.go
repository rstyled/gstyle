package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/runtimeguard"
	"github.com/rstyled/gstyle/domain/runtimeguard/interval"
)

// Flow tests keep checking the raw missing-bound facts. That internal query
// does not establish that an invalid index reaches a source access.
func potentialNegativeAccesses(t *testing.T, source string) []interval.Access {
	t.Helper()
	var out []interval.Access
	for _, function := range runtimeTestGraphs(t, source) {
		report, err := runtimeguard.Analyze(&function.Pdg)
		if err != nil {
			t.Fatal(err)
		}
		var accesses []interval.Access
		for _, site := range report.Accesses {
			accesses = append(accesses, interval.Access{Node: uint32(site.Node), Symbol: uint32(site.Symbol), Index: site.Index, UpperChecked: site.Value.UpperChecked, Bounds: site.Value.Bounds()})
		}
		out = append(out, interval.Select(accesses)...)
	}
	return out
}

func TestIndexEvidenceRequiresInvalidRange(t *testing.T) {
	cases := []struct {
		name              string
		value             runtimeguard.Value
		negative, literal bool
	}{
		{"unknown", runtimeguard.Value{HasExtent: true, Extent: 2}, false, false},
		{"negative", runtimeguard.Value{HasUpper: true, Upper: -1, HasExtent: true, Extent: 2}, true, true},
		{"too high", runtimeguard.Value{HasLower: true, Lower: 2, HasExtent: true, Extent: 2}, false, true},
		{"partly valid", runtimeguard.Value{HasLower: true, Lower: -1, HasUpper: true, Upper: 2, HasExtent: true, Extent: 2}, false, false},
		{"valid", runtimeguard.Value{HasLower: true, Lower: 0, HasUpper: true, Upper: 1, HasExtent: true, Extent: 2}, false, false},
		{"unknown extent", runtimeguard.Value{HasLower: true, Lower: 2}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			site := runtimeguard.Access{Index: true, Literal: true, Value: tc.value}
			if got := knownNegativeIndex(site); got != tc.negative {
				t.Fatalf("negative=%v want %v", got, tc.negative)
			}
			if got := literalIndexRisk(site); got != tc.literal {
				t.Fatalf("literal=%v want %v", got, tc.literal)
			}
		})
	}
}
