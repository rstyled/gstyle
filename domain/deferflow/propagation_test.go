package deferflow

import "testing"

func TestChildPanicRetainsItsOwnGoexitDuringParentUnwind(t *testing.T) {
	parent := exitFixture[string]()
	startActiveExit(t, &parent)
	child := ExitResult[string]{Kind: PanicExit, Panic: PanicValue[string]{Value: "child", Kind: OrdinaryPanicValue}, ResumeGoexit: true, Complete: true}
	if err := parent.CompletePropagation(child); err != nil {
		t.Fatal(err)
	}
	pending := parent.Exit()
	if pending.Kind != PanicExit || !pending.ResumeGoexit || pending.Panic.Value != "child" {
		t.Fatalf("lost child continuation: %+v", pending)
	}
	result, err := ResolveInvocation(pending, ExitResult[string]{Kind: ReturnExit, Complete: true}, true)
	if err != nil || result.Kind != GoexitExit {
		t.Fatalf("recovered child must resume Goexit: %+v %v", result, err)
	}
}
func TestInvalidChildPropagationIsAtomic(t *testing.T) {
	for _, child := range []ExitResult[string]{{Kind: PanicExit}, {Kind: NoExit, Complete: true}, {Kind: PanicExit, Complete: true, Panic: PanicValue[string]{Kind: 99}}} {
		parent := exitFixture[string]()
		startActiveExit(t, &parent)
		before := parent.StateKey()
		if err := parent.CompletePropagation(child); err == nil {
			t.Fatal("accepted invalid child")
		}
		if parent.StateKey() != before {
			t.Fatal("invalid child changed parent")
		}
	}
}
func TestResolveInvocationModes(t *testing.T) {
	for _, tc := range []struct {
		pending, child, want ExitKind
		barrier, recover     bool
	}{
		{ReturnExit, ReturnExit, ReturnExit, false, false},
		{PanicExit, ReturnExit, PanicExit, false, false},
		{PanicExit, ReturnExit, ReturnExit, false, true},
		{PanicExit, ReturnExit, GoexitExit, true, true},
		{GoexitExit, PanicExit, PanicExit, true, false},
		{PanicExit, GoexitExit, GoexitExit, false, false},
		{ReturnExit, AbortExit, AbortExit, false, false},
	} {
		got, err := ResolveInvocation(ExitResult[int]{Kind: tc.pending, ResumeGoexit: tc.barrier}, ExitResult[int]{Kind: tc.child, Complete: true}, tc.recover)
		if err != nil || got.Kind != tc.want {
			t.Fatalf("%+v: %+v %v", tc, got, err)
		}
	}
}
