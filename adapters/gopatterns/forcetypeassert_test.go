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
			fn, fs, _ := parseNilErrFunc(t, `package p
func f(v any){`+test.body+`}
func wrap(s string)(string,bool){return s,true}`)
			if hits := findForceTypeAssertHits(fn, fs); len(hits) != test.want {
				t.Fatalf("got %+v, want %d unchecked assertions", hits, test.want)
			}
		})
	}
}

func TestForceTypeAssertEmptyInterfaces(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       []string
	}{
		{"any", `_ = v.(any)`, []string{`v.(any)`}},
		{"literal interface", `_ = v.(interface{})`, []string{`v.(interface{})`}},
		{"alias", `_ = v.(EmptyAlias)`, []string{`v.(EmptyAlias)`}},
		{"defined interface", `_ = v.(Empty)`, []string{`v.(Empty)`}},
		{"parenthesized type", `_ = v.((any))`, []string{`v.((any))`}},
		{"checked any", `x,ok:=v.(any);_,_=x,ok`, nil},
		{"checked assignment", `var x any;var ok bool;x,ok=v.(any);_,_=x,ok`, nil},
		{"checked literal declaration", `var x,ok=v.(interface{});_,_=x,ok`, nil},
		{"checked alias", `x,ok:=v.(EmptyAlias);_,_=x,ok`, nil},
		{"checked defined interface", `x,ok:=v.(Empty);_,_=x,ok`, nil},
		{"nested unchecked", `_ = v.(any).(string)`, []string{`v.(any).(string)`, `v.(any)`}},
		{"checked outer", `x,ok:=v.(any).(string);_,_=x,ok`, []string{`v.(any)`}},
		{"type switch", `switch v.(type){case any:}`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn, fs, _ := parseNilErrFunc(t, `package p
type EmptyAlias = any
type Empty interface{}
func f(v any){`+test.body+`}`)
			hits := findForceTypeAssertHits(fn, fs)
			if len(hits) != len(test.want) {
				t.Fatalf("got %+v, want %v", hits, test.want)
			}
			for i, want := range test.want {
				if hits[i].ExprText != want {
					t.Errorf("assertion %d: got %q, want %q", i, hits[i].ExprText, want)
				}
			}
		})
	}
}
