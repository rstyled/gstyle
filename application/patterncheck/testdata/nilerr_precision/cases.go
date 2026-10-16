package precision

import (
	"errors"
	"io"
)

type Result struct {
	Err     error
	Pending bool
}

func MobyBad(fetch func() error) ([]string, error) {
	if err := fetch(); err != nil {
		return []string{}, nil
	}
	return nil, nil
}
func MobySafe(fetch func() error) ([]string, error) {
	if err := fetch(); err != nil {
		return []string{}, err
	}
	return nil, nil
}
func LoggedBad(r *Result) error {
	if r.Err != nil {
		println(r.Err)
		return nil
	}
	return nil
}
func FieldBad(r *Result) error {
	if r.Err != nil {
		return nil
	}
	return nil
}
func FallbackUnknown(lookup func() (string, error), hostname string) (string, error) {
	_, err := lookup()
	if err != nil {
		return hostname, nil
	}
	return hostname, nil
}
func EOFSafe(r *Result) (int, error) {
	if r.Err != nil {
		if errors.Is(r.Err, io.EOF) && r.Pending {
			r.Pending = false
			return 1, nil
		}
		return 0, r.Err
	}
	return 0, nil
}
func ExpectedSafe(r *Result, expected error) error {
	if r.Err != nil {
		if err := checkExpected(r.Err, expected); err != nil {
			return err
		}
		return nil
	}
	return errors.New("expected a failure")
}
func checkExpected(got, want error) error {
	if !errors.Is(got, want) {
		return got
	}
	return nil
}
func SiblingBad(err error, handled bool) error {
	if err != nil {
		if handled {
			if checkExpected(err, err) != nil {
				return err
			}
			return nil
		}
		return nil
	}
	return nil
}

type Data struct{ N int }

func ZeroAggregateBad(err error) (Data, error) {
	if err != nil {
		return Data{N: 0}, nil
	}
	return Data{}, nil
}
func NamedNilBad(err error) (result error) {
	if err != nil {
		return
	}
	return err
}
func LocalNilBad(err error) error {
	if err != nil {
		var result error
		return result
	}
	return err
}
func DeferredSafe(cause error) (err error) {
	defer func() { err = cause }()
	if cause != nil {
		println(cause)
		return nil
	}
	return nil
}
func UnreachableSafe(err error) error {
	if err != nil {
		println(err)
		if false {
			return nil
		}
		return err
	}
	return nil
}
func CorrelatedSafe(err error, b bool) error {
	if err != nil {
		if b {
			return err
		}
		if !b {
			return err
		}
		return nil
	}
	return nil
}
func ReceiverRepairUnknown(p *Result) error {
	if p.Err != nil {
		repairResult(p)
		return nil
	}
	return nil
}
func repairResult(p *Result) { p.Err = nil }
func RedundantCheckBad(err error) error {
	if err != nil {
		if err == nil {
			return err
		}
		return nil
	}
	return nil
}

type pointerError struct{}

func (*pointerError) Error() string { return "failure" }
func TypedNilSafe(cause error) error {
	if cause != nil {
		var p *pointerError
		var out error = p
		return out
	}
	return nil
}
func GenericZeroUnknown[T error](cause error) (out T) {
	if cause != nil {
		return
	}
	return
}

type Payload struct{ V any }

func BoxedFallbackUnknown(cause error) (Payload, error) {
	if cause != nil {
		return Payload{V: 0}, nil
	}
	return Payload{}, nil
}

func InverseBad(err error) error {
	if err == nil {
		return nil
	}
	return nil
}
