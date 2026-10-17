package deferflow

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

type unwindAction uint8

const (
	actionReturn unwindAction = iota
	actionRecover
	actionPanic
	actionGoexit
	actionIndirectRecover
	actionInternalRecover
)

type unwindTrace struct {
	Events []string
	Kind   ExitKind
	Panic  string
}

func panicLabel(value any) string {
	if value == nil {
		return "nil"
	}
	switch value.(type) {
	case *runtime.PanicNilError:
		return "nil-error"
	case *int:
		return "typed-nil"
	default:
		return fmt.Sprint(value)
	}
}
func indirectRecovery() any { return recover() }
func nativeUnwind(initial int, actions []unwindAction) unwindTrace {
	done := make(chan unwindTrace, 1)
	go func() {
		trace := unwindTrace{}
		returned := false
		defer func() {
			value := recover()
			trace.Kind = ReturnExit
			if value != nil {
				trace.Kind = PanicExit
				trace.Panic = panicLabel(value)
			} else if !returned {
				trace.Kind = GoexitExit
			}
			done <- trace
		}()
		func() {
			for i, action := range actions {
				defer func(i int, action unwindAction) {
					label := strconv.Itoa(i)
					trace.Events = append(trace.Events, "invoke:"+label)
					switch action {
					case actionRecover:
						trace.Events = append(trace.Events, "recover:"+panicLabel(recover()))
					case actionPanic:
						panic("panic-" + label)
					case actionGoexit:
						runtime.Goexit()
					case actionIndirectRecover:
						trace.Events = append(trace.Events, "indirect:"+panicLabel(indirectRecovery()))
					case actionInternalRecover:
						defer func() { trace.Events = append(trace.Events, "internal:"+panicLabel(recover())) }()
						panic("internal-" + label)
					}
				}(i, action)
			}
			switch initial {
			case 1:
				panic("initial")
			case 2:
				runtime.Goexit()
			case 3:
				panic(nil)
			case 4:
				var pointer *int
				panic(pointer)
			}
		}()
		returned = true
	}()
	return <-done
}
func exitFixture[T any]() Machine[T] {
	return Machine[T]{sites: map[pdg.Ref]Site{1: {Registration: 1, Invocation: 2, Captures: []Capture{{Slot: Slot{Kind: "argument", Position: 1, Mode: "value"}, Source: 3}}}}}
}
func modeledUnwind(t *testing.T, initial int, actions []unwindAction) unwindTrace {
	t.Helper()
	machine := exitFixture[string]()
	for i := range actions {
		if err := machine.Register(1, []string{strconv.Itoa(i)}); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	switch initial {
	case 0:
		err = machine.BeginNormalExit()
	case 1:
		err = machine.BeginPanicExit(PanicValue[string]{Value: "initial", Kind: OrdinaryPanicValue})
	case 2:
		err = machine.BeginGoexit()
	case 3:
		err = machine.BeginPanicExit(PanicValue[string]{Kind: NilPanicError})
	case 4:
		err = machine.BeginPanicExit(PanicValue[string]{Value: "typed-nil", Kind: OrdinaryPanicValue})
	}
	if err != nil {
		t.Fatal(err)
	}
	trace := unwindTrace{}
	for {
		call, found, err := machine.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !found {
			break
		}
		label := call.Captures[0].Value
		index, _ := strconv.Atoi(label)
		trace.Events = append(trace.Events, "invoke:"+label)
		switch actions[index] {
		case actionReturn:
			err = machine.Complete()
		case actionRecover:
			value, found, recoverErr := machine.CompleteRecovered()
			err = recoverErr
			text := "nil"
			if found {
				text = modeledPanicLabel(value)
			}
			trace.Events = append(trace.Events, "recover:"+text)
		case actionPanic:
			err = machine.CompletePanic(PanicValue[string]{Value: "panic-" + label, Kind: OrdinaryPanicValue})
		case actionGoexit:
			err = machine.CompleteGoexit()
		case actionIndirectRecover:
			trace.Events = append(trace.Events, "indirect:nil")
			err = machine.Complete()
		case actionInternalRecover:
			trace.Events = append(trace.Events, "internal:internal-"+label)
			err = machine.Complete()
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	result := machine.Exit()
	if !result.Complete {
		t.Fatal("finished stack does not complete exit")
	}
	trace.Kind = result.Kind
	if result.Kind == PanicExit {
		trace.Panic = modeledPanicLabel(result.Panic)
	}
	return trace
}
func modeledPanicLabel(value PanicValue[string]) string {
	if value.Kind == NilPanicError {
		return "nil-error"
	}
	return value.Value
}
func TestExceptionalUnwindMatchesNativeGo(t *testing.T) {
	t.Setenv("GODEBUG", "panicnil=0")
	checked := 0
	for length := 0; length <= 3; length++ {
		layouts := 1
		for range length {
			layouts *= 6
		}
		for layout := 0; layout < layouts; layout++ {
			digits := layout
			actions := make([]unwindAction, length)
			for i := range actions {
				actions[i] = unwindAction(digits % 6)
				digits /= 6
			}
			for initial := 0; initial < 5; initial++ {
				got, want := modeledUnwind(t, initial, actions), nativeUnwind(initial, actions)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("initial=%d actions=%v got=%+v want=%+v", initial, actions, got, want)
				}
				checked++
			}
		}
	}
	t.Logf("checked %d native panic/recovery/Goexit traces", checked)
}
