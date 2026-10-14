package query

import (
	"math"
	"testing"
)

func TestReferenceBoundaries(t *testing.T) {
	for _, test := range []struct {
		ref   Ref
		count int
		index int
		ok    bool
	}{
		{0, 0, 0, false}, {0, 2, 0, false}, {1, -1, 0, false}, {1, 0, 0, false},
		{1, 2, 0, true}, {2, 2, 1, true}, {3, 2, 0, false},
	} {
		index, ok := ReferenceIndex(test.ref, test.count)
		if index != test.index || ok != test.ok {
			t.Fatalf("%+v: got (%d, %v)", test, index, ok)
		}
	}
	if uint64(math.MaxInt) > math.MaxUint32 {
		ref := Ref(math.MaxUint32)
		index, ok := ReferenceIndex(ref, math.MaxInt)
		if !ok || index != int(uint64(ref)-1) {
			t.Fatal("largest reference does not resolve")
		}
	}
}

func TestEvidencePresenceAndPolicies(t *testing.T) {
	records := make([]Capability, 256)
	for i := range records {
		records[i] = Capability{Description: Text(i + 1), Status: Status(i)}
	}
	for i, want := range records {
		ref := Ref(i + 1)
		got, ok := EvidenceAt(records, ref)
		if !ok || got != want || EvidenceStatus(records, ref) != want.Status {
			t.Fatalf("slot %d: %+v, %v", i, got, ok)
		}
		coverage := i == 1 || i == 3 || i == 4
		dependency := i == 1 || i == 4
		if CoverageSupported(want.Status) != coverage || DependencyAdmitted(records, ref) != dependency {
			t.Fatalf("status %d policy mismatch", i)
		}
	}
	for _, ref := range []Ref{0, 257, math.MaxUint32} {
		got, ok := EvidenceAt(records, ref)
		if ok || got != (Capability{}) || EvidenceStatus(records, ref) != Unknown || DependencyAdmitted(records, ref) {
			t.Fatalf("invalid ref %d is admitted", ref)
		}
	}
	if _, ok := EvidenceAt(nil, 1); ok {
		t.Fatal("nil table produces a record")
	}
	if got, ok := EvidenceAt(records, 1); !ok || got.Status != Unknown {
		t.Fatal("present unknown record is confused with absence")
	}
}

func TestKnownFalseIsNotMissingOrApproximate(t *testing.T) {
	for status := 0; status < 256; status++ {
		for _, payload := range []bool{false, true} {
			fact := BooleanFact{Status: Status(status), Value: payload}
			value, known := BooleanValue(fact)
			if status == int(Known) {
				if !known || value != payload {
					t.Fatalf("known payload lost: %+v", fact)
				}
			} else if known || value {
				t.Fatalf("unavailable payload trusted: %+v", fact)
			}
			if got := KnownFalse(fact); got != (status == int(Known) && !payload) {
				t.Fatalf("known-false gate: %+v", fact)
			}
		}
	}
}
