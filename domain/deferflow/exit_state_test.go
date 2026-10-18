package deferflow

import (
	"runtime"
	"testing"
)

func startActiveExit(t *testing.T, machine *Machine[string]) {
	t.Helper()
	if err := machine.Register(1, []string{"capture"}); err != nil {
		t.Fatal(err)
	}
	if err := machine.BeginPanicExit(PanicValue[string]{Value: "old", Kind: OrdinaryPanicValue}); err != nil {
		t.Fatal(err)
	}
	if _, found, err := machine.Next(); err != nil || !found {
		t.Fatalf("active invocation=%v,%v", found, err)
	}
}
func finishExit[T any](t *testing.T, machine *Machine[T]) ExitResult[T] {
	t.Helper()
	if _, found, err := machine.Next(); err != nil || found {
		t.Fatalf("finished invocation=%v,%v", found, err)
	}
	result := machine.Exit()
	if !result.Complete {
		t.Fatal("exit is incomplete")
	}
	return result
}
func TestExceptionalExitForksAndStateKeys(t *testing.T) {
	base := exitFixture[string]()
	startActiveExit(t, &base)
	left, right := base, base
	before := right.StateKey()
	value, recovered, err := left.CompleteRecovered()
	if err != nil || !recovered || value.Value != "old" {
		t.Fatalf("recover=%+v,%v,%v", value, recovered, err)
	}
	if right.StateKey() != before || left.StateKey() == before {
		t.Fatal("recovery mutates a fork or is absent from its key")
	}
	if err := right.CompletePanic(PanicValue[string]{Value: "new", Kind: OrdinaryPanicValue}); err != nil {
		t.Fatal(err)
	}
	if left.StateKey() == right.StateKey() {
		t.Fatal("different exit continuations share a scheduling key")
	}
	if result := finishExit(t, &left); result.Kind != ReturnExit {
		t.Fatalf("recovery exit=%+v", result)
	}
	if result := finishExit(t, &right); result.Kind != PanicExit || result.Panic.Value != "new" {
		t.Fatalf("replacement panic=%+v", result)
	}
	noncomparable := exitFixture[map[string]int]()
	if err := noncomparable.BeginPanicExit(PanicValue[map[string]int]{Value: map[string]int{"value": 1}}); err != nil {
		t.Fatal(err)
	}
	key := noncomparable.StateKey()
	if key != noncomparable.StateKey() {
		t.Fatal("noncomparable value prevents state comparison")
	}
}
func TestExceptionalContinuationPropagatesGoexitBarrier(t *testing.T) {
	child := exitFixture[string]()
	if err := child.Register(1, []string{"panic"}); err != nil {
		t.Fatal(err)
	}
	if err := child.BeginGoexit(); err != nil {
		t.Fatal(err)
	}
	if _, found, err := child.Next(); err != nil || !found {
		t.Fatal(err)
	}
	if err := child.CompletePanic(PanicValue[string]{Value: "new", Kind: OrdinaryPanicValue}); err != nil {
		t.Fatal(err)
	}
	result := finishExit(t, &child)
	if result.Kind != PanicExit || !result.ResumeGoexit {
		t.Fatalf("lost Goexit continuation=%+v", result)
	}
	parent := exitFixture[string]()
	if err := parent.Register(1, []string{"recover"}); err != nil {
		t.Fatal(err)
	}
	if err := parent.BeginPropagation(result); err != nil {
		t.Fatal(err)
	}
	if _, found, err := parent.Next(); err != nil || !found {
		t.Fatal(err)
	}
	value, recovered, err := parent.CompleteRecovered()
	if err != nil || !recovered || value.Value != "new" {
		t.Fatalf("parent recovery=%+v,%v,%v", value, recovered, err)
	}
	if final := finishExit(t, &parent); final.Kind != GoexitExit {
		t.Fatalf("recovery cancels Goexit=%+v", final)
	}
	done := make(chan bool, 1)
	go func() {
		recovered, continued := false, false
		defer func() { done <- recover() == nil && recovered && !continued }()
		func() {
			defer func() { recovered = recover() == "nested" }()
			func() { defer func() { panic("nested") }(); runtime.Goexit() }()
			continued = true
		}()
		continued = true
	}()
	if !<-done {
		t.Fatal("native caller recovery bypasses a pending Goexit")
	}
}
func TestExceptionalExitRejectsInvalidTransitionsAtomically(t *testing.T) {
	machine := exitFixture[string]()
	before := machine.StateKey()
	invalid := PanicValue[string]{Kind: PanicKind(99)}
	for _, operation := range []func() error{
		func() error { return machine.CompletePanic(PanicValue[string]{}) },
		machine.CompleteGoexit,
		func() error { _, _, err := machine.CompleteRecovered(); return err },
		func() error { return machine.BeginPanicExit(invalid) },
		func() error { return machine.BeginPropagation(ExitResult[string]{Kind: PanicExit}) },
		func() error { return machine.BeginPropagation(ExitResult[string]{Kind: ReturnExit, Complete: true}) },
		func() error {
			return machine.BeginPropagation(ExitResult[string]{Kind: PanicExit, Panic: invalid, Complete: true})
		},
	} {
		if err := operation(); err == nil {
			t.Fatal("invalid transition accepted")
		}
		if machine.StateKey() != before {
			t.Fatal("failed transition changes scheduling state")
		}
	}
	startActiveExit(t, &machine)
	active := machine.StateKey()
	if err := machine.CompletePanic(invalid); err == nil || machine.StateKey() != active {
		t.Fatal("invalid panic changes active call")
	}
	if err := machine.BeginGoexit(); err == nil || machine.StateKey() != active {
		t.Fatal("begin exit overlaps unwind")
	}
	machine.Abort()
	if result := machine.Exit(); !result.Complete || result.Kind != AbortExit {
		t.Fatalf("abort=%+v", result)
	}
	if err := machine.CompleteGoexit(); err == nil {
		t.Fatal("aborted invocation resumes")
	}
}
