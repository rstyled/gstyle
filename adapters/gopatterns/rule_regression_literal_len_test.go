package gopatterns

import "testing"

func TestAuditLiteralSliceLengthGuard(t *testing.T) {
	src := `package p;func f(i int)int{if i<0 || i>=len([]int{1,2}){return 0};return []int{1,2}[i]}`
	if got := literalIndexTestCandidates(t, src); len(got) != 0 {
		t.Fatalf("safe literal-length guard: got %d findings", len(got))
	}
}
