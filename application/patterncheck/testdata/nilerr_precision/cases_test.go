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
