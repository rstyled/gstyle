package gopatterns

import "testing"

func TestErrorCallChecks(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		checked    bool
	}{
		{"direct comparison", `if produce() != nil { return nil }`, true},
		{"reversed comparison", `if nil == produce() { return nil }`, true},
		{"parenthesized comparison", `if (produce()) != nil { return nil }`, true},
		{"parenthesized return", `return (produce())`, true},
		{"parenthesized assignment", `err := (produce()); if err != nil { return err }`, true},
		{"grouped multi assignment", `a,err := 1,(produce()); _ = a; if err != nil { return err }`, true},
		{"discarded", `produce()`, false},
		{"parenthesized discard", `_ = (produce())`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p;func f()error{`+tc.body+`;return nil};func produce()error{return nil}`)
			sites := findErrorCheckSites(fn, fs, info)
			if len(sites) != 1 || sites[0].Checked != tc.checked {
				t.Fatalf("sites=%+v, want checked=%v", sites, tc.checked)
			}
		})
	}
}

func TestAliasedErrorCall(t *testing.T) {
	for _, tc := range []struct {
		body    string
		checked bool
	}{
		{`produce()`, false},
		{`err:=produce();if err!=nil{return err}`, true},
		{`_,err:=pair();if err!=nil{return err}`, true},
	} {
		fn, fs, info := parseNilErrFunc(t, `package p;type Err = error;func f()error{`+tc.body+`;return nil};func produce()Err{return nil};func pair()(int,Err){return 0,nil}`)
		sites := findErrorCheckSites(fn, fs, info)
		if len(sites) != 1 || sites[0].Checked != tc.checked {
			t.Fatalf("%s: sites=%+v, want checked=%v", tc.body, sites, tc.checked)
		}
	}
}
