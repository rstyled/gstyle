package gopatterns

import "testing"

func TestNilErrLoggedCompletion(t *testing.T) {
	for _, source := range []string{
		`func f(cause error)(err error){defer func(){err=cause}();if cause!=nil{println(cause);return nil};return nil}`,
		`func f(cause error)error{clear:=func(){cause=nil};if cause!=nil{println(cause);clear();return nil};return nil}`,
		`func f(err error)error{if err!=nil{println(err);if false{return nil};return err};return nil}`,
	} {
		fn, fs, info := parseNilErrFunc(t, "package p;"+source)
		if got := findNilErrHits(fn, fs, info); len(got) != 0 {
			t.Errorf("false warning for %s: %+v", source, got)
		}
	}
}
