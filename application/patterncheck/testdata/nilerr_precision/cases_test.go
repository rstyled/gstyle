package precision

import (
	"errors"
	"io"
	"testing"
)

func TestNilReturnBehavior(t *testing.T) {
	cause := errors.New("fetch failed")
	fetch := func() error { return cause }
	if value, err := MobyBad(fetch); err != nil || len(value) != 0 {
		t.Fatalf("Moby bad: value=%v,error=%v", value, err)
	}
	if _, err := MobySafe(fetch); !errors.Is(err, cause) {
		t.Fatalf("Moby safe: %v", err)
	}
	for _, call := range []func(*Result) error{FieldBad, LoggedBad} {
		if err := call(&Result{Err: cause}); err != nil {
			t.Fatal(err)
		}
	}
	if err := SiblingBad(cause, false); err != nil {
		t.Fatal(err)
	}
	if got, err := FallbackUnknown(func() (string, error) { return "", cause }, "localhost"); got != "localhost" || err != nil {
		t.Fatalf("fallback=%q,error=%v", got, err)
	}
	r := &Result{Err: io.EOF, Pending: true}
	if got, err := EOFSafe(r); got != 1 || err != nil || r.Pending {
		t.Fatalf("buffered result=%d,error=%v,pending=%v", got, err, r.Pending)
	}
	if _, err := EOFSafe(r); !errors.Is(err, io.EOF) {
		t.Fatalf("EOF not preserved: %v", err)
	}
	if err := ExpectedSafe(&Result{Err: cause}, cause); err != nil {
		t.Fatal(err)
	}
	if err := ExpectedSafe(&Result{Err: cause}, io.EOF); !errors.Is(err, cause) {
		t.Fatalf("unexpected failure swallowed: %v", err)
	}
}

func TestRegressionBehavior(t *testing.T) {
	cause := errors.New("failed")
	value, err := ZeroAggregateBad(cause)
	if value != (Data{}) || err != nil {
		t.Fatalf("zero result=%v,error=%v", value, err)
	}
	for _, call := range []func(error) error{NamedNilBad, LocalNilBad, RedundantCheckBad, InverseBad} {
		if err := call(cause); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range []func(error) error{DeferredSafe, UnreachableSafe} {
		if !errors.Is(call(cause), cause) {
			t.Fatal("failure was not propagated")
		}
	}
	for _, b := range []bool{true, false} {
		if !errors.Is(CorrelatedSafe(cause, b), cause) {
			t.Fatal("branch lost failure")
		}
	}
	p := &Result{Err: cause}
	if err := ReceiverRepairUnknown(p); err != nil || p.Err != nil {
		t.Fatal("repair did not complete")
	}
}

func TestNilnessBoundaries(t *testing.T) {
	cause := errors.New("failed")
	if TypedNilSafe(cause) == nil {
		t.Fatal("typed nil pointer became nil error")
	}
	var boxed error = GenericZeroUnknown[*pointerError](cause)
	if boxed == nil {
		t.Fatal("generic pointer zero became nil interface")
	}
	if GenericZeroUnknown[error](cause) != nil {
		t.Fatal("generic interface zero is not nil")
	}
	value, err := BoxedFallbackUnknown(cause)
	if err != nil || value.V == nil {
		t.Fatalf("boxed fallback=%v,error=%v", value, err)
	}
}
