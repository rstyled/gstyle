package gopatterns

import "testing"

func TestNilErrAlternativeOperationOrder(t *testing.T) {
	for _, init := range []string{`retryErr,n:=retry(),number()`, `n,retryErr:=number(),retry()`} {
		fn, fs, info := parseNilErrFunc(t, `package p;func f(cause error)error{if cause!=nil{println(cause);if `+init+`;retryErr!=nil{_=n;return retryErr};return nil};return nil};func retry()error{return nil};func number()int{return 0}`)
		if got := findNilErrHits(fn, fs, info); len(got) != 0 {
			t.Errorf("%s: unexpected finding %+v", init, got)
		}
	}
}
