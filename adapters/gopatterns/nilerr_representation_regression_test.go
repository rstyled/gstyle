package gopatterns

import "testing"

func TestNilErrZeroResultRepresentation(t *testing.T) {
	for _, value := range []string{`Data{}`, `Data{N:0}`, `Data{0}`} {
		fn, fs, info := parseNilErrFunc(t, `package p;func f(err error)(Data,error){if err!=nil{return `+value+`,nil};return Data{},nil};type Data struct{N int}`)
		if got := findNilErrHits(fn, fs, info); len(got) != 1 {
			t.Errorf("%s loses detection: %+v", value, got)
		}
	}
}
func TestNilErrPromotedErrorHandling(t *testing.T) {
	for _, pair := range [][2]string{{"p.Err", "p.Err"}, {"p.Err", "p.R.Err"}, {"p.R.Err", "p.Err"}, {"p.R.Err", "p.R.Err"}} {
		fn, fs, info := parseNilErrFunc(t, `package p;func f(p *Outer)error{if `+pair[0]+`!=nil{if classify(`+pair[1]+`){return nil};return `+pair[0]+`};return nil};type R struct{Err error};type Outer struct{*R};func classify(error)bool{return true}`)
		if got := findNilErrHits(fn, fs, info); len(got) != 0 {
			t.Errorf("check %s,handle %s: %+v", pair[0], pair[1], got)
		}
	}
}
