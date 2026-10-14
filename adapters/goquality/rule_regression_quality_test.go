package goquality

import (
	"context"
	"github.com/rstyled/gstyle/domain/quality"
	"testing"
)

func TestAuditLintIgnoreExtraction(t *testing.T) {
	root := qualitySourceModule(t, []byte("package p\n//lint:ignore fn_params Public API contract\nfunc f(a,b,c,d,e int)int{return a+b+c+d+e}\n"))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range report.Diagnostics {
		if d.RuleID == "fn_params" {
			t.Fatal("lint:ignore was lost before evaluation")
		}
	}
}
func TestAuditMutatingStdlibEffects(t *testing.T) {
	cases := []struct{ name, src string }{
		{"sort", `package p;import "sort";func f(x []int){sort.Ints(x);_=x;_=len(x)}`},
		{"slices", `package p;import "slices";func f(x []int){slices.Sort(x);_=x;_=len(x)}`},
		{"maps", `package p;import "maps";func f(x,y map[string]int){maps.Copy(x,y);_=x;_=len(x)}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := qualitySourceModule(t, []byte(tc.src))
			facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			report, err := quality.Evaluate(facts, quality.DefaultConfig())
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range report.Functions {
				if f.DensityMilli == 0 && f.Complete {
					t.Fatalf("observable mutation classified pure: %+v", f)
				}
			}
		})
	}
}
