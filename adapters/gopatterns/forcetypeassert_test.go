package gopatterns

import "testing"

func TestForceTypeAssertCommaOK(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       int
	}{
		{"short assignment", `s,ok:=v.(string);_,_=s,ok`, 0},
		{"assignment", `var s string;var ok bool;s,ok=v.(string);_,_=s,ok`, 0},
		{"declaration", `var s,ok=v.(string);_,_=s,ok`, 0},
		{"parenthesized assignment", `s,ok:=((v.(string)));_,_=s,ok`, 0},
		{"parenthesized declaration", `var s,ok=((v.(string)));_,_=s,ok`, 0},
		{"wrapped assignment", `s,ok:=wrap(v.(string));_,_=s,ok`, 1},
		{"wrapped declaration", `var s,ok=wrap(v.(string));_,_=s,ok`, 1},
		{"nested assignment", `s,ok:=v.(interface{M()}).(interface{N()});_,_=s,ok`, 1},
		{"nested declaration", `var s,ok=v.(interface{M()}).(interface{N()});_,_=s,ok`, 1},
		{"separate results", `s,ok:=v.(string),true;_,_=s,ok`, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p
func f(v any){`+test.body+`}
func wrap(s string)(string,bool){return s,true}`)
			if hits := findForceTypeAssertHits(fn, fs, info); len(hits) != test.want {
				t.Fatalf("got %+v, want %d unchecked assertions", hits, test.want)
			}
		})
	}
}
