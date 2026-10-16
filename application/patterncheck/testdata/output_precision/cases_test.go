package precision

import (
	"errors"
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
