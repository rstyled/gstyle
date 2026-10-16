package precision

import (
	"fmt"
	"io"
)

func OutputDirectBad(w io.Writer) error  { fmt.Fprintln(w, "result"); return nil }
func OutputDirectSafe(w io.Writer) error { _, err := fmt.Fprintln(w, "result"); return err }
func OutputLocalBad(w io.Writer) error   { message := "result"; fmt.Fprintln(w, message); return nil }
func OutputLocalSafe(w io.Writer) error {
	message := "result"
	_, err := fmt.Fprintln(w, message)
	return err
}
func OutputBranchBad(w io.Writer) error {
	if w != nil {
		fmt.Fprintln(w, "result")
		return nil
	}
	return io.ErrClosedPipe
}
func OutputBranchSafe(w io.Writer) error {
	if w != nil {
		_, err := fmt.Fprintln(w, "result")
		return err
	}
	return io.ErrClosedPipe
}
func OutputPanicSafe(w io.Writer) error { fmt.Fprintln(w, never()); return nil }
func never() string                     { panic("before output") }
func OutputExistingFailureSafe(w io.Writer, cause error) error {
	if cause != nil {
		fmt.Fprintln(w, cause)
		return cause
	}
	return nil
}

func OutputFalseSafe(w io.Writer) error {
	if false {
		fmt.Fprintln(w, "result")
		return nil
	}
	return nil
}
func OutputContradictionSafe(w io.Writer) error {
	if w == nil {
		if w != nil {
			fmt.Fprintln(w, "result")
			return nil
		}
	}
	return nil
}

var boxedWrites int

type nilWriter struct{}

func (*nilWriter) Write([]byte) (int, error) { boxedWrites++; return 0, io.ErrClosedPipe }
func OutputBoxedNilBad() error {
	var p *nilWriter
	var w io.Writer = p
	if w != nil {
		fmt.Fprintln(w, "result")
		return nil
	}
	return io.ErrClosedPipe
}
func OutputBoxedNilSafe() error {
	var p *nilWriter
	var w io.Writer = p
	if w != nil {
		_, err := fmt.Fprintln(w, "result")
		return err
	}
	return io.ErrClosedPipe
}
func OutputClearedInterfaceSafe(w io.Writer) error {
	w = nil
	if w != nil {
		fmt.Fprintln(w, "result")
		return nil
	}
	return nil
}
func OutputZeroInterfaceSafe() error {
	var w io.Writer
	if w != nil {
		fmt.Fprintln(w, "result")
		return nil
	}
	return nil
}

func OutputNilGuardWriterSafe(w io.Writer) error {
	if w == nil {
		fmt.Fprintln(w, "result")
		return nil
	}
	return nil
}
func OutputLiteralNilWriterSafe() error             { fmt.Fprintln(nil, "result"); return nil }
func OutputDeclaredNilWriterSafe() error            { var w io.Writer; fmt.Fprintln(w, "result"); return nil }
func OutputAssignedNilWriterSafe(w io.Writer) error { w = nil; fmt.Fprintln(w, "result"); return nil }
func OutputReturnPanicSafe(w io.Writer) (int, error) {
	fmt.Fprintln(w, "result")
	return neverInt(), nil
}
func neverInt() int { panic("after output") }
func OutputReturnIndexPanicSafe(w io.Writer) (int, error) {
	s := []int{}
	fmt.Fprintln(w, "result")
	return s[0], nil
}

func OutputDeferredPanicSafe(w io.Writer) error {
	defer func() { panic("after output") }()
	fmt.Fprintln(w, "result")
	return nil
}
func OutputNormalDeferUnknown(w io.Writer, done *bool) error {
	defer func() { *done = true }()
	fmt.Fprintln(w, "result")
	return nil
}
func OutputDeferredErrorSafe(w io.Writer) (err error) {
	defer func() { err = io.ErrClosedPipe }()
	fmt.Fprintln(w, "result")
	return nil
}
func OutputAssignmentNameBad(w io.Writer) error {
	n := 0
	_ = n
	n, _ = fmt.Fprintln(w, "result")
	return nil
}
func OutputSliceTargetUnknown(w io.Writer, s []int) error {
	s[0], _ = fmt.Fprintln(w, "result")
	return nil
}
func OutputPointerTargetUnknown(w io.Writer, p *int) error {
	*p, _ = fmt.Fprintln(w, "result")
	return nil
}

func OutputAssignmentNameSafe(w io.Writer) error {
	n := 0
	_ = n
	var err error
	n, err = fmt.Fprintln(w, "result")
	return err
}

// These have the same output tail. The operation contract differs.
func OutputRequiredContractUnknown(w io.Writer) error {
	fmt.Fprintln(w, "result")
	return nil
}
func OutputStatusContractUnknown(w io.Writer, completed *bool) error {
	*completed = true
	fmt.Fprintln(w, "result")
	return nil
}
