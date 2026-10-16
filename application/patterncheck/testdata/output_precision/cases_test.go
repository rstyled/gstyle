package precision

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

type failingWriter struct{ writes int }

func (w *failingWriter) Write([]byte) (int, error) { w.writes++; return 0, io.ErrClosedPipe }
func TestWriteBehavior(t *testing.T) {
	cases := []struct {
		name string
		call func(io.Writer) error
		lost bool
	}{
		{"OutputDirectBad", OutputDirectBad, true}, {"OutputDirectSafe", OutputDirectSafe, false},
		{"OutputLocalBad", OutputLocalBad, true}, {"OutputLocalSafe", OutputLocalSafe, false},
		{"OutputBranchBad", OutputBranchBad, true}, {"OutputBranchSafe", OutputBranchSafe, false},
		{"OutputAssignmentNameBad", OutputAssignmentNameBad, true}, {"OutputAssignmentNameSafe", OutputAssignmentNameSafe, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &failingWriter{}
			err := tc.call(w)
			if w.writes != 1 || (err == nil) != tc.lost {
				t.Fatalf("writes=%d,error=%v,want lost=%v", w.writes, err, tc.lost)
			}
		})
	}
}
func TestOutputPanicDoesNotReachWrite(t *testing.T) {
	w := &failingWriter{}
	defer func() {
		if recover() == nil || w.writes != 0 {
			t.Fatalf("expected panic before write;writes=%d", w.writes)
		}
	}()
	OutputPanicSafe(w)
}
func TestExistingFailureStaysFailure(t *testing.T) {
	w := &failingWriter{}
	cause := errors.New("existing failure")
	if err := OutputExistingFailureSafe(w, cause); w.writes != 1 || !errors.Is(err, cause) {
		t.Fatalf("writes=%d,error=%v", w.writes, err)
	}
}

func TestUnreachableWritesDoNotRun(t *testing.T) {
	for _, call := range []func(io.Writer) error{OutputFalseSafe, OutputContradictionSafe} {
		w := &failingWriter{}
		if err := call(w); err != nil || w.writes != 0 {
			t.Fatalf("writes=%d,error=%v", w.writes, err)
		}
	}
}

func TestBoxedNilIsANonNilInterface(t *testing.T) {
	boxedWrites = 0
	if err := OutputBoxedNilBad(); err != nil || boxedWrites != 1 {
		t.Fatalf("bad write: count=%d,error=%v", boxedWrites, err)
	}
	boxedWrites = 0
	if err := OutputBoxedNilSafe(); !errors.Is(err, io.ErrClosedPipe) || boxedWrites != 1 {
		t.Fatalf("safe write: count=%d,error=%v", boxedWrites, err)
	}
	w := &failingWriter{}
	if err := OutputClearedInterfaceSafe(w); err != nil || w.writes != 0 {
		t.Fatalf("cleared interface: writes=%d,error=%v", w.writes, err)
	}
	if err := OutputZeroInterfaceSafe(); err != nil {
		t.Fatal(err)
	}
}

func TestNilWriterCannotReportSuccess(t *testing.T) {
	calls := []func(io.Writer) error{
		func(io.Writer) error { return OutputNilGuardWriterSafe(nil) },
		func(io.Writer) error { return OutputLiteralNilWriterSafe() },
		func(io.Writer) error { return OutputDeclaredNilWriterSafe() },
		OutputAssignedNilWriterSafe,
	}
	for i, call := range calls {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			w := &failingWriter{}
			defer func() {
				if recover() == nil || w.writes != 0 {
					t.Fatalf("expected panic before write;writes=%d", w.writes)
				}
			}()
			call(w)
		})
	}
}
func TestReturnPanicCannotReportSuccess(t *testing.T) {
	for i, call := range []func(io.Writer) (int, error){OutputReturnPanicSafe, OutputReturnIndexPanicSafe} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			w := &failingWriter{}
			defer func() {
				if recover() == nil || w.writes != 1 {
					t.Fatalf("expected panic after write;writes=%d", w.writes)
				}
			}()
			call(w)
		})
	}
}

func TestOutputCompletionBoundaries(t *testing.T) {
	t.Run("normal defer completes", func(t *testing.T) {
		w := &failingWriter{}
		done := false
		if err := OutputNormalDeferUnknown(w, &done); err != nil || w.writes != 1 || !done {
			t.Fatalf("writes=%d,error=%v,done=%v", w.writes, err, done)
		}
	})
	t.Run("defer replaces error", func(t *testing.T) {
		w := &failingWriter{}
		if err := OutputDeferredErrorSafe(w); !errors.Is(err, io.ErrClosedPipe) || w.writes != 1 {
			t.Fatalf("writes=%d,error=%v", w.writes, err)
		}
	})
	for name, call := range map[string]func(io.Writer) error{
		"deferred panic": OutputDeferredPanicSafe,
		"slice target":   func(w io.Writer) error { return OutputSliceTargetUnknown(w, nil) },
		"pointer target": func(w io.Writer) error { return OutputPointerTargetUnknown(w, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			w := &failingWriter{}
			defer func() {
				if recover() == nil || w.writes != 1 {
					t.Fatalf("expected panic after write;writes=%d", w.writes)
				}
			}()
			call(w)
		})
	}
	t.Run("valid targets complete", func(t *testing.T) {
		w := &failingWriter{}
		s := make([]int, 1)
		p := 0
		if err := OutputSliceTargetUnknown(w, s); err != nil {
			t.Fatal(err)
		}
		if err := OutputPointerTargetUnknown(w, &p); err != nil {
			t.Fatal(err)
		}
		if w.writes != 2 {
			t.Fatalf("writes=%d", w.writes)
		}
	})
}
