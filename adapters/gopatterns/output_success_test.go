package gopatterns

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestOutputSuccessEvidence(t *testing.T) {
	cases := []struct {
		name, body string
		report     bool
	}{
		{"success return", `fmt.Fprintln(w,err);return nil`, true},
		{"blank error", `_,_=fmt.Fprintln(w,err);return nil`, true},
		{"nil branch result", `if err!=nil{fmt.Fprintln(w,err);return nil};return nil`, true},

		{"original error", `if err!=nil{fmt.Fprintln(w,err);return err};return nil`, false},
		{"nested diagnostic", `if err!=nil{if w!=nil{fmt.Fprintln(w,err)};return err};return nil`, false},
		{"unknown return", `fmt.Fprintln(w,err);return err`, false},
		{"unknown call between write and return", `fmt.Fprintln(w,err);other();return nil`, false},
		{"nil is shadowed", `nil:=err;fmt.Fprintln(w,err);return nil`, false},
		{"nil variable", `var result error;fmt.Fprintln(w,err);return result`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `package fixture;import("fmt";"io");func other(){};func f(w io.Writer,err error)error{` + tc.body + `}`
			file, fset, info, _ := typeCheck(t, src)
			sites, success := findErrorCheckFacts(findFunc(file, "f"), fset, info)
			discarded := 0
			for _, site := range sites {
				if site.DiscardedOutput {
					discarded++
				}
			}
			if discarded != 1 {
				t.Fatalf("raw discarded calls=%d,want 1", discarded)
			}
			if (len(success) == 1) != tc.report {
				t.Fatalf("success sites=%v,want report=%v", success, tc.report)
			}
		})
	}
}

func TestOutputSuccessReturnShapes(t *testing.T) {
	cases := []struct {
		name, source string
		report       bool
	}{
		{"tuple", `func f(w io.Writer)(int,error){fmt.Fprintln(w,"x");return 0,nil}`, true},
		{"named result", `func f(w io.Writer)(err error){fmt.Fprintln(w,"x");return nil}`, true},
		{"deferred named replacement", `func f(w io.Writer)(err error){defer func(){err=fmt.Errorf("failed")}();fmt.Fprintln(w,"x");return nil}`, false},
		{"void best effort", `func f(w io.Writer){fmt.Fprintln(w,"x")}`, false},
		{"exit callback", `func f(w io.Writer,err error,exit func(int)){if err!=nil{fmt.Fprintln(w,err);exit(1)}}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, info, _ := typeCheck(t, `package fixture;import("fmt";"io");`+tc.source)
			_, success := findErrorCheckFacts(findFunc(file, "f"), fset, info)
			if (len(success) == 1) != tc.report {
				t.Fatalf("success sites=%v,want report=%v", success, tc.report)
			}
		})
	}
}

type failedDiagnosticWriter struct{ writes int }

func (w *failedDiagnosticWriter) Write([]byte) (int, error) { w.writes++; return 0, io.ErrClosedPipe }
func diagnosticThenFailure(w io.Writer, err error) error {
	if err != nil {
		fmt.Fprintln(w, err)
		return err
	}
	return nil
}
func outputThenSuccess(w io.Writer) error { fmt.Fprintln(w, "result"); return nil }

func TestFailedWriteBehavior(t *testing.T) {
	cause := errors.New("read failed")
	diagnostic := &failedDiagnosticWriter{}
	if got := diagnosticThenFailure(diagnostic, cause); diagnostic.writes != 1 || !errors.Is(got, cause) {
		t.Fatalf("diagnostic writes=%d,result=%v", diagnostic.writes, got)
	}
	output := &failedDiagnosticWriter{}
	if got := outputThenSuccess(output); output.writes != 1 || got != nil {
		t.Fatalf("output writes=%d,result=%v", output.writes, got)
	}
}

func TestRuntimeTypeParameterIsNotAnInterfaceValue(t *testing.T) {
	file, _, info, _ := typeCheck(t, `package p;func f[T ~int](value T){}`)
	parameter := findFunc(file, "f").Type.TypeParams.List[0].Names[0]
	if got := runtimeTypeKind(info.Defs[parameter].Type()); got != "" {
		t.Fatalf("type parameter classified as runtime %q", got)
	}
}
