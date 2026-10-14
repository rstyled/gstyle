package gopatterns

import "testing"

func TestAuditNilErrExtended(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"simple swallowed", `if err!=nil{return nil};return nil`, 1},
		{"propagated", `if err!=nil{return err};return nil`, 0},
		{"handled", `if err!=nil{println(err);return nil};return nil`, 0},
		{"parenthesized", `if (err!=nil){return nil};return nil`, 1},
		{"negated", `if !(err==nil){return nil};return nil`, 1},
		{"shadow nil is error value", `nil:=err;if err!=nil{return nil};return nil`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `package p;func f(err error)error{` + tc.body + `}`
			fn, fset, info := parseNilErrFunc(t, src)
			if got := findNilErrHits(fn, fset, info); len(got) != tc.want {
				t.Fatalf("got %d findings,want %d", len(got), tc.want)
			}
		})
	}
}
