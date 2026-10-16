package gopatterns

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestOutputFailureEvidence(t *testing.T) {
	cases := []struct {
		name, body string
		excluded   bool
	}{
		{"returns original error", `if err != nil { fmt.Fprintln(w, err); return err }; return nil`, true},
		{"nested diagnostic", `if err != nil { if w != nil { fmt.Fprintln(w, err) }; return err }; return nil`, true},
		{"success return", `if err != nil { fmt.Fprintln(w, err); return nil }; return nil`, false},
		{"unconditional write", `fmt.Fprintln(w, err); return nil`, false},
		{"reset error", `if err != nil { fmt.Fprintln(w, err); err = nil; return err }; return nil`, false},
		{"reset before write", `if err != nil { err = nil; fmt.Fprintln(w, err); return err }; return nil`, false},
		{"range resets error", `if err != nil { for _, err = range []error{nil} {}; fmt.Fprintln(w, err); return err }; return nil`, false},
		{"alternate success", `if err != nil { fmt.Fprintln(w, err); if w == nil { return nil }; return err }; return nil`, false},
		{"address escapes", `change := func(p *error) { *p = nil }; if err != nil { change(&err); fmt.Fprintln(w, err); return err }; return nil`, false},
		{"captured outside guard", `change := func() { err = nil }; if err != nil { change(); fmt.Fprintln(w, err); return err }; return nil`, false},
		{"branch escapes", `for { if err != nil { fmt.Fprintln(w, err); break; return err }; break }; return nil`, false},
		{"shadowed nil", `nil := err; if err != nil { fmt.Fprintln(w, err); return err }; return nil`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package fixture; import (\"fmt\"; \"io\"); func f(w io.Writer, err error) error {" + tc.body + "}"
			file, fset, info, _ := typeCheck(t, source)
			sites, excluded := findErrorCheckFacts(findFunc(file, "f"), fset, info)
			discarded := 0
			for _, site := range sites {
				if site.DiscardedOutput {
					discarded++
				}
			}
			if discarded != 1 {
				t.Fatalf("raw discarded calls = %d, want 1", discarded)
			}
			if (len(excluded) == 1) != tc.excluded {
				t.Fatalf("excluded = %v, want %v", excluded, tc.excluded)
			}
		})
	}
}

func TestOutputFailureReturnShapes(t *testing.T) {
	cases := []struct {
		name, source string
		excluded     bool
	}{
		{"tuple", `func f(w io.Writer, err error) (any,error) { if err != nil { fmt.Fprintln(w,err); return nil,err }; return nil,nil }`, true},
		{"initializer", `func f(w io.Writer) error { if err := fmt.Errorf("cause"); err != nil { fmt.Fprintln(w,err); return err }; return nil }`, true},
		{"named deferred overwrite", `func f(w io.Writer, cause error) (err error) { defer func(){ err=nil }(); err=cause; if err != nil { fmt.Fprintln(w,err); return err }; return nil }`, false},
		{"global binding", `var err error; func f(w io.Writer) error { if err != nil { fmt.Fprintln(w,err); return err }; return nil }`, false},
		{"exit callback", `func f(w io.Writer, err error, exit func(int)) { if err != nil { fmt.Fprintln(w,err); exit(1) } }`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, info, _ := typeCheck(t, `package fixture; import ("fmt"; "io"); `+tc.source)
			_, excluded := findErrorCheckFacts(findFunc(file, "f"), fset, info)
			if (len(excluded) == 1) != tc.excluded {
				t.Fatalf("excluded = %v, want %v", excluded, tc.excluded)
			}
		})
	}
}

type failedDiagnosticWriter struct{ writes int }

func (w *failedDiagnosticWriter) Write([]byte) (int, error) { w.writes++; return 0, io.ErrClosedPipe }

// This is the reported source shape. Force the diagnostic write to fail and
// observe the result independently of the analyzer's source facts.
func diagnosticThenFailure(w io.Writer, err error) error {
	if err != nil {
		fmt.Fprintln(w, err)
		return err
	}
	return nil
}

func TestDiagnosticWriteFailureStillReturnsCause(t *testing.T) {
	cause := errors.New("read failed")
	writer := &failedDiagnosticWriter{}
	got := diagnosticThenFailure(writer, cause)
	if writer.writes != 1 || !errors.Is(got, cause) {
		t.Fatalf("writes=%d result=%v, want original failure", writer.writes, got)
	}
}
