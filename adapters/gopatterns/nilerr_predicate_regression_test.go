package gopatterns

import "testing"

func TestNilErrRedundantNilPredicate(t *testing.T) {
	for _, source := range []string{
		`func f(err error)error{if err!=nil{if err==nil{return err};return nil};return nil}`,
		`func f(p *R)error{if p.Err!=nil{if p.Err==nil{return p.Err};return nil};return nil}`,
	} {
		fn, fs, info := parseNilErrFunc(t, "package p;"+source+`;type R struct{Err error}`)
		if got := findNilErrHits(fn, fs, info); len(got) != 1 {
			t.Errorf("lost error missed for %s: %+v", source, got)
		}
	}
}
