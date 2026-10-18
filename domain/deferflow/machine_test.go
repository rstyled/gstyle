package deferflow

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func machineFixture() Machine[int] {
	return Machine[int]{sites: map[pdg.Ref]Site{1: {Registration: 1, Invocation: 2, Captures: []Capture{{Slot: Slot{Kind: "argument", Position: 1, Mode: "value"}, Source: 3}}}}}
}
func nativeDeferredValues(values []int) (out []int) {
	for _, value := range values {
		defer func(snapshot int) { out = append(out, snapshot) }(value)
	}
	return
}
func drainMachine(t *testing.T, machine Machine[int]) []int {
	t.Helper()
	if err := machine.BeginNormalExit(); err != nil {
		t.Fatal(err)
	}
	var out []int
	for {
		call, found, err := machine.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !found {
			return out
		}
		if call.Registration != 1 || call.Node != 2 || len(call.Captures) != 1 {
			t.Fatalf("invocation=%+v", call)
		}
		out = append(out, call.Captures[0].Value)
		if err := machine.Complete(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestNormalUnwindAgainstGoDefer(t *testing.T) {
	checked := 0
	for length := 0; length <= 6; length++ {
		count := 1
		for range length {
			count *= 3
		}
		for layout := 0; layout < count; layout++ {
			machine := machineFixture()
			var values []int
			digits := layout
			for range length {
				value := digits % 3
				digits /= 3
				values = append(values, value)
				operand := []int{value}
				if err := machine.Register(1, operand); err != nil {
					t.Fatal(err)
				}
				operand[0] = 99 // Register saves this dynamic occurrence, not the caller's slice.
			}
			got, want := drainMachine(t, machine), nativeDeferredValues(values)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("values=%v got=%v want=%v", values, got, want)
			}
			checked++
		}
	}
	t.Logf("checked %d dynamic registration traces against Go defers", checked)
}
func TestNormalUnwindForksOwnDynamicFrames(t *testing.T) {
	original := machineFixture()
	if err := original.Register(1, []int{1}); err != nil {
		t.Fatal(err)
	}
	left, right := original, original
	if err := left.Register(1, []int{2}); err != nil {
		t.Fatal(err)
	}
	if err := right.Register(1, []int{3}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		machine Machine[int]
		want    []int
	}{{left, []int{2, 1}}, {right, []int{3, 1}}, {original, []int{1}}} {
		if got := drainMachine(t, tc.machine); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("got=%v want=%v", got, tc.want)
		}
	}
}
func TestNormalUnwindReturnedInputsAreOwned(t *testing.T) {
	machine := machineFixture()
	site, _ := machine.Site(1)
	site.Captures[0].Source = 99
	if err := machine.Register(1, []int{7}); err != nil {
		t.Fatal(err)
	}
	copy := machine
	if err := machine.BeginNormalExit(); err != nil {
		t.Fatal(err)
	}
	invocation, _, err := machine.Next()
	if err != nil {
		t.Fatal(err)
	}
	if invocation.Captures[0].Source != 3 {
		t.Fatal("Site exposes mutable inventory")
	}
	invocation.Captures[0].Value = 42
	if got := drainMachine(t, copy); !reflect.DeepEqual(got, []int{7}) {
		t.Fatalf("returned inputs mutate shared stack: %v", got)
	}
}
func TestNormalUnwindPhaseBoundaries(t *testing.T) {
	machine := machineFixture()
	if _, _, err := machine.Next(); err == nil {
		t.Fatal("invocation starts before exit")
	}
	if err := machine.Complete(); err == nil {
		t.Fatal("completion without invocation")
	}
	if err := machine.Register(99, []int{1}); err == nil {
		t.Fatal("unknown registration")
	}
	if err := machine.Register(1, nil); err == nil {
		t.Fatal("missing capture values")
	}
	if err := machine.Register(1, []int{1}); err != nil {
		t.Fatal(err)
	}
	if err := machine.BeginNormalExit(); err != nil {
		t.Fatal(err)
	}
	if err := machine.BeginNormalExit(); err == nil {
		t.Fatal("repeated exit")
	}
	if err := machine.Register(1, []int{1}); err == nil {
		t.Fatal("registration during unwind")
	}
	if _, found, err := machine.Next(); err != nil || !found {
		t.Fatalf("next found=%v err=%v", found, err)
	}
	if _, _, err := machine.Next(); err == nil {
		t.Fatal("next invocation overlaps active call")
	}
	if err := machine.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := machine.Complete(); err == nil {
		t.Fatal("duplicate invocation completion")
	}
	if _, found, err := machine.Next(); err != nil || found {
		t.Fatalf("empty unwind found=%v err=%v", found, err)
	}
	if _, _, err := machine.Next(); err == nil {
		t.Fatal("unwind restarts after exit")
	}
}
func TestAbortDoesNotUnwind(t *testing.T) {
	for _, active := range []bool{false, true} {
		machine := machineFixture()
		if err := machine.Register(1, []int{1}); err != nil {
			t.Fatal(err)
		}
		if active {
			if err := machine.BeginNormalExit(); err != nil {
				t.Fatal(err)
			}
			if _, _, err := machine.Next(); err != nil {
				t.Fatal(err)
			}
		}
		machine.Abort()
		if machine.top != nil {
			t.Fatal("abort retains saved frames")
		}
		if _, _, err := machine.Next(); err == nil {
			t.Fatal("abort invokes a pending defer")
		}
		if err := machine.Complete(); err == nil {
			t.Fatal("abort retains an active invocation")
		}
	}
}

func TestSavedValuesCopyValuesAndPreserveReferences(t *testing.T) {
	type payload struct {
		Number int
		Cell   *int
	}
	cell := 1
	value := payload{Number: 2, Cell: &cell}
	machine := Machine[payload]{sites: machineFixture().sites}
	if err := machine.Register(1, []payload{value}); err != nil {
		t.Fatal(err)
	}
	value.Number = 99
	cell = 7
	if err := machine.BeginNormalExit(); err != nil {
		t.Fatal(err)
	}
	call, found, err := machine.Next()
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	saved := call.Captures[0].Value
	if saved.Number != 2 || saved.Cell != &cell || *saved.Cell != 7 {
		t.Fatalf("saved=%+v cell=%d", saved, *saved.Cell)
	}
}

func TestSchedulingStateKeyIncludesPendingFramesAndPhase(t *testing.T) {
	base := machineFixture()
	before := base.StateKey()
	left, right := base, base
	if left.StateKey() != right.StateKey() {
		t.Fatal("fork changes scheduling state")
	}
	if err := left.Register(1, []int{1}); err != nil {
		t.Fatal(err)
	}
	if left.StateKey() == before || left.StateKey() == right.StateKey() {
		t.Fatal("pending frame is absent from state key")
	}
	afterRegistration := left.StateKey()
	if err := left.BeginNormalExit(); err != nil {
		t.Fatal(err)
	}
	if left.StateKey() == afterRegistration {
		t.Fatal("exit phase is absent from state key")
	}
}

func TestSharedIndexKeepsExecutionAndReturnedMetadataIndependent(t *testing.T) {
	fixture := machineFixture()
	index := Index{sites: []Site{fixture.sites[1]}, lookup: fixture.sites}
	left, right := NewMachineWithIndex[int](index), NewMachineWithIndex[int](index)
	sites := index.Sites()
	sites[0].Captures[0].Source = 99
	site, found := left.Site(1)
	if !found || site.Captures[0].Source != 3 {
		t.Fatal("returned metadata changes shared index")
	}
	if err := left.Register(1, []int{7}); err != nil {
		t.Fatal(err)
	}
	if got := drainMachine(t, left); !reflect.DeepEqual(got, []int{7}) {
		t.Fatalf("left=%v", got)
	}
	if got := drainMachine(t, right); len(got) != 0 {
		t.Fatalf("shared execution state: %v", got)
	}
}
