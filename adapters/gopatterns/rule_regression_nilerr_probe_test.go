package gopatterns

import "testing"

func TestAuditNilErrProbes(t *testing.T) {
	cases := []struct{ name, body string }{
		{"one branch swallows", `if err!=nil{if choose{return nil};return err};return nil`},
		{"unused closure is not handling", `if err!=nil{_ = func(){println(err)};return nil};return nil`},
		{"shadow argument is not checked error", `if err!=nil{func(err string){println(err)}("other");return nil};return nil`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `package p;func f(err error,choose bool)error{` + tc.body + `}`
			fn, fset, info := parseNilErrFunc(t, src)
			if got := findNilErrHits(fn, fset, info); len(got) != 1 {
				t.Fatalf("got %d findings,want 1", len(got))
			}
		})
	}
}
