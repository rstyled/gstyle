package gopatterns

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditForceAssertMultiAssignment(t *testing.T) {
	src := `package p;func f(x any){a,b:=x.(int),2;_,_=a,b}`
	fn, fs, info := parseNilErrFunc(t, src)
	if got := findForceTypeAssertHits(fn, fs, info); len(got) != 1 {
		t.Fatalf("panic-producing assertion: got %d findings,want 1", len(got))
	}
}
func TestAuditDeviantChecks(t *testing.T) {
	cases := []struct {
		name, body string
		want       bool
	}{
		{"assigned and discarded", `err:=produce();_=err`, false},
		{"non-error output discarded", `_,err:=pair();if err!=nil{println(err)}`, true},
		{"propagated", `return produce()`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `package p;func f()error{` + tc.body + `;return nil};func produce()error{return nil};func pair()(int,error){return 1,nil}`
			fn, fs, info := parseNilErrFunc(t, src)
			sites := findErrorCheckSites(fn, fs, info)
			if len(sites) != 1 || sites[0].Checked != tc.want {
				t.Fatalf("got %+v,want checked=%v", sites, tc.want)
			}
		})
	}
}
func TestAuditGuardSuppression(t *testing.T) {
	cases := []struct{ kind, source string }{
		{"nil_guard", `func f(p *int)int{if p!=nil{return 0};return *p}`},
		{"negative_index", `func f(s []int,i int)int{if i>=len(s){return 0};return s[i]}`},
		{"literal_index", `func f(i int)int{return []int{1,2}[i]}`},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			root := t.TempDir()
			os.WriteFile(filepath.Join(root, "go.mod"), []byte("module audit\n\ngo 1.26\n"), 0600)
			os.WriteFile(filepath.Join(root, "p.go"), []byte("package audit\n//lint:ignore "+tc.kind+" intentional contract\n"+tc.source), 0600)
			extraction, err := (&FixExtractor{}).Extract(context.Background(), root, []string{"."}, SourceOverlay{})
			if err != nil {
				t.Fatal(err)
			}
			n := len(extraction.NilGuards) + len(extraction.NegativeIndices) + len(extraction.LiteralIndices)
			if n != 0 {
				t.Fatalf("ignored %s still emits %d findings", tc.kind, n)
			}
		})
	}
}

func TestErrorCheckResultBindingAndOrder(t *testing.T) {
	cases := []struct {
		body string
		want []bool
	}{
		{`left,right:=produce(),produce();_=left;if right!=nil{println(right)}`, []bool{false, true}},
		{`err:=produce();if err!=nil{println(err)};err=produce();_=err`, []bool{true, false}},
	}
	for _, tc := range cases {
		fn, fs, info := parseNilErrFunc(t, `package p;func f(){`+tc.body+`};func produce()error{return nil}`)
		sites := findErrorCheckSites(fn, fs, info)
		if len(sites) != len(tc.want) {
			t.Fatalf("sites: %+v", sites)
		}
		for i, site := range sites {
			if site.Checked != tc.want[i] {
				t.Fatalf("site %d: checked=%v want=%v", i, site.Checked, tc.want[i])
			}
		}
	}
}
