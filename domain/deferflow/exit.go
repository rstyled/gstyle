package deferflow

import "fmt"

// ExitKind describes the current function's continuation after its defers.
type ExitKind uint8

const (
	NoExit ExitKind = iota
	ReturnExit
	PanicExit
	GoexitExit
	AbortExit
)

type PanicKind uint8

const (
	UnknownPanicValue PanicKind = iota
	OrdinaryPanicValue
	NilPanicError
)

// PanicValue carries the consumer's evaluated value abstraction. NilPanicError
// represents Go's normalized panic(nil) payload, not a recovered nil interface.
// Unknown is explicit when the consumer cannot establish the representation.
type PanicValue[T any] struct {
	Value T
	Kind  PanicKind
}

type exitState[T any] struct {
	kind   ExitKind
	panic  PanicValue[T]
	goexit bool
}

type ExitResult[T any] struct {
	Kind         ExitKind
	Panic        PanicValue[T]
	ResumeGoexit bool
	Complete     bool
}

func (m *Machine[T]) BeginPanicExit(value PanicValue[T]) error {
	if err := validPanic(value); err != nil {
		return err
	}
	return m.beginExit(exitState[T]{kind: PanicExit, panic: value})
}

func (m *Machine[T]) BeginGoexit() error {
	return m.beginExit(exitState[T]{kind: GoexitExit, goexit: true})
}

// BeginPropagation transfers a completed exceptional continuation to a caller.
// A normal return resumes its caller's body and must not start caller unwind.
func (m *Machine[T]) BeginPropagation(result ExitResult[T]) error {
	if !result.Complete {
		return fmt.Errorf("exceptional continuation is not complete")
	}
	switch result.Kind {
	case PanicExit:
		if err := validPanic(result.Panic); err != nil {
			return err
		}
		return m.beginExit(exitState[T]{kind: PanicExit, panic: result.Panic, goexit: result.ResumeGoexit})
	case GoexitExit:
		return m.BeginGoexit()
	default:
		return fmt.Errorf("continuation does not unwind its caller")
	}
}

func (m Machine[T]) Exit() ExitResult[T] {
	if m.exit == nil {
		return ExitResult[T]{}
	}
	return ExitResult[T]{Kind: m.exit.kind, Panic: m.exit.panic, ResumeGoexit: m.exit.goexit, Complete: m.phase == finished}
}

// CompletePanic reports a panic that escapes the active deferred call, after
// that call's own defers. A recovered internal panic is ordinary Complete.
// The escaped panic supersedes the parent's panic at this invocation boundary.
func (m *Machine[T]) CompletePanic(value PanicValue[T]) error {
	if err := validPanic(value); err != nil {
		return err
	}
	if !m.active {
		return fmt.Errorf("no deferred invocation is active")
	}
	return m.completeExit(exitState[T]{kind: PanicExit, panic: value, goexit: m.exit.goexit})
}

func (m *Machine[T]) CompleteGoexit() error {
	return m.completeExit(exitState[T]{kind: GoexitExit, goexit: true})
}

// CompletePropagation applies a completed child exit at an active invocation.
// A child panic can carry its own suspended Goexit into the parent's unwind.
func (m *Machine[T]) CompletePropagation(result ExitResult[T]) error {
	if !m.active || !result.Complete {
		return fmt.Errorf("propagation requires an active invocation and completed child")
	}
	return m.propagateExit(result)
}

// CompleteRecovered reports that the active deferred call directly recovered
// its inherited panic and returned. This is a consumer-proved call outcome; it
// does not infer direct recover eligibility or execute the deferred body.
func (m *Machine[T]) CompleteRecovered() (PanicValue[T], bool, error) {
	if !m.active {
		return PanicValue[T]{}, false, fmt.Errorf("no deferred invocation is active")
	}
	if m.exit.kind != PanicExit {
		return PanicValue[T]{}, false, m.Complete()
	}
	value := m.exit.panic
	next := exitState[T]{kind: ReturnExit}
	if m.exit.goexit {
		next = exitState[T]{kind: GoexitExit, goexit: true}
	}
	return value, true, m.completeExit(next)
}

func (m *Machine[T]) beginExit(state exitState[T]) error {
	if m.phase != running {
		return fmt.Errorf("function exit already started")
	}
	m.exit = &state
	m.phase = unwinding
	return nil
}

func (m *Machine[T]) completeExit(state exitState[T]) error {
	if !m.active {
		return fmt.Errorf("no deferred invocation is active")
	}
	m.exit = &state
	m.active = false
	return nil
}

func (m *Machine[T]) propagateExit(result ExitResult[T]) error {
	switch result.Kind {
	case ReturnExit:
		return m.Complete()
	case PanicExit:
		return m.completePropagatedPanic(result)
	case GoexitExit:
		return m.CompleteGoexit()
	case AbortExit:
		m.Abort()
		return nil
	default:
		return fmt.Errorf("invalid child exit")
	}
}

func (m *Machine[T]) completePropagatedPanic(result ExitResult[T]) error {
	if err := validPanic(result.Panic); err != nil {
		return err
	}
	return m.completeExit(exitState[T]{kind: PanicExit, panic: result.Panic, goexit: m.exit.goexit || result.ResumeGoexit})
}

func validPanic[T any](value PanicValue[T]) error {
	if value.Kind > NilPanicError {
		return fmt.Errorf("invalid panic value representation")
	}
	return nil
}

// ResolveInvocation composes one proved child outcome with a pending unwind.
// It does not infer recovery eligibility or execute the child's body.
func ResolveInvocation[T any](pending, completed ExitResult[T], recovered bool) (ExitResult[T], error) {
	if err := validPendingExit(pending); err != nil {
		return ExitResult[T]{}, err
	}
	m := Machine[T]{phase: unwinding, active: true, exit: &exitState[T]{kind: pending.Kind, panic: pending.Panic, goexit: pending.ResumeGoexit || pending.Kind == GoexitExit}}
	if !recovered {
		err := m.CompletePropagation(completed)
		return m.Exit(), err
	}
	if !completed.Complete || completed.Kind != ReturnExit {
		return ExitResult[T]{}, fmt.Errorf("recovery requires a completed normal return")
	}
	_, _, err := m.CompleteRecovered()
	return m.Exit(), err
}

func validPendingExit[T any](pending ExitResult[T]) error {
	switch pending.Kind {
	case ReturnExit, PanicExit, GoexitExit:
		return validPanic(pending.Panic)
	default:
		return fmt.Errorf("invalid pending unwind")
	}
}
