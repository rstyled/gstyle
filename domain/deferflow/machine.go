package deferflow

import (
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type phase uint8

const (
	running phase = iota
	unwinding
	finished
)

type savedFrame[T any] struct {
	site     Site
	values   []T
	previous *savedFrame[T]
}

// Invocation contains one dynamic registration's saved inputs. T follows Go
// value-copy semantics: references in a value still name the same storage.
type Invocation[T any] struct {
	Registration, Node pdg.Ref
	Captures           []SavedCapture[T]
	Spread, Tuple      bool
}

type SavedCapture[T any] struct {
	Capture
	Value T
}

// Machine schedules one function's deferred invocations during exit. Frames are immutable so
// a value copy forks the stack in O(1). A consumer must fork its own value state,
// evaluate captures before Register, and apply call effects before Complete.
// Deferred call outcomes update the exit continuation. Body execution, recovery
// eligibility and memory/result effects remain the consumer's responsibility.
type Machine[T any] struct {
	exit   *exitState[T]
	sites  map[pdg.Ref]Site
	top    *savedFrame[T]
	phase  phase
	active bool
}

func NewMachine[T any](graph *pdg.Graph) (Machine[T], error) {
	index, err := NewIndex(graph)
	if err != nil {
		return Machine[T]{}, err
	}
	return NewMachineWithIndex[T](index), nil
}

// NewMachineWithIndex starts fresh dynamic state using immutable capture facts.
// A copied index shares metadata only; frames and unwind state remain local.
func NewMachineWithIndex[T any](index Index) Machine[T] {
	return Machine[T]{sites: index.lookup}
}

// Site returns an owned capture inventory, not the machine's private record.
func (m Machine[T]) Site(ref pdg.Ref) (Site, bool) {
	site, found := m.sites[ref]
	site.Captures = slices.Clone(site.Captures)
	return site, found
}

func (m *Machine[T]) Register(ref pdg.Ref, values []T) error {
	if m.phase != running {
		return fmt.Errorf("cannot register a defer after function exit starts")
	}
	site, found := m.sites[ref]
	if !found {
		return fmt.Errorf("unknown deferred registration")
	}
	if len(values) != len(site.Captures) {
		return fmt.Errorf("deferred capture value count does not match inventory")
	}
	m.top = &savedFrame[T]{site: site, values: slices.Clone(values), previous: m.top}
	return nil
}

func (m *Machine[T]) BeginNormalExit() error {
	return m.beginExit(exitState[T]{kind: ReturnExit})
}

// Next starts the most recent remaining invocation. Complete must acknowledge
// that invocation before another can start. An empty stack completes the exit.
func (m *Machine[T]) Next() (Invocation[T], bool, error) {
	if m.phase != unwinding {
		return Invocation[T]{}, false, fmt.Errorf("deferred unwind is not active")
	}
	if m.active {
		return Invocation[T]{}, false, fmt.Errorf("previous deferred invocation is not complete")
	}
	if m.top == nil {
		m.phase = finished
		return Invocation[T]{}, false, nil
	}
	frame := m.top
	m.top = frame.previous
	m.active = true
	return savedInvocation(frame), true, nil
}

func (m *Machine[T]) Complete() error {
	if !m.active {
		return fmt.Errorf("no deferred invocation is active")
	}
	m.active = false
	return nil
}

// Abort models an exit that does not run any remaining defers, such as os.Exit.
func (m *Machine[T]) Abort() {
	*m = Machine[T]{sites: m.sites, phase: finished, exit: &exitState[T]{kind: AbortExit}}
}

func (m Machine[T]) StateKey() StateKey {
	return StateKey{top: m.top, exit: m.exit, phase: m.phase, active: m.active}
}

// StateKey compares saved scheduling state within one machine context. A
// consumer's mutable memory/value state is separate and needs its own key.
// The opaque top is always a comparable pointer to an immutable saved frame.
type StateKey struct {
	top    any
	exit   any
	phase  phase
	active bool
}

func savedInvocation[T any](frame *savedFrame[T]) Invocation[T] {
	captures := make([]SavedCapture[T], len(frame.values))
	for i, value := range frame.values {
		captures[i] = SavedCapture[T]{Capture: frame.site.Captures[i], Value: value}
	}
	return Invocation[T]{Registration: frame.site.Registration, Node: frame.site.Invocation, Captures: captures, Spread: frame.site.Spread, Tuple: frame.site.Tuple}
}
