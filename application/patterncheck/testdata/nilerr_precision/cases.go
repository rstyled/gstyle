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
