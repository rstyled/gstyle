package gopatterns

import (
	"github.com/rstyled/gstyle/domain/runtimeguard"
	"testing"
)

func TestSecondReview(t *testing.T) {
	t.Run("guarded promoted field", func(t *testing.T) {
		src := `package p; type Inner struct{Value int};type Outer struct{*Inner};func f(p Outer,q *int)int{if q==nil || p.Inner==nil{return 0};return p.Value}`
		if got := unprovenPointerAccesses(t, src); len(got) != 0 {
			t.Fatalf("safe promoted access: %d findings", len(got))
		}
	})
	t.Run("copied nil guard", func(t *testing.T) {
		src := `package p;func f(p,q *int)int{r:=p;if q==nil || r==nil{return 0};return *p}`
		if got := unprovenPointerAccesses(t, src); len(got) != 0 {
			t.Fatalf("safe aliased pointer: %d findings", len(got))
		}
	})
	t.Run("pointer array index", func(t *testing.T) {
		src := `package p;func f(s *[3]int,i int)int{if i>=len(s){return 0};return s[i]}`
		site := reviewAccess(t, src)
		if !site.Index || !site.Value.UpperChecked || site.Value.HasLower {
			t.Fatalf("pointer-array access: %+v", site)
		}
	})
	t.Run("safe reverse variable bound", func(t *testing.T) {
		src := `package p;func f(i int)int{lo:=0;if lo>i || i>=2{return 0};return []int{1,2}[i]}`
		site := reviewAccess(t, src)
		if !site.Value.HasLower || !site.Value.HasUpper || site.Value.Lower != 0 || site.Value.Upper != 1 {
			t.Fatalf("reversed bounds: %+v", site.Value)
		}
	})
	t.Run("safe equality variable bound", func(t *testing.T) {
		src := `package p;func f(i int)int{want:=1;if want!=i{return 0};return []int{1,2}[i]}`
		site := reviewAccess(t, src)
		if !site.Value.HasLower || !site.Value.HasUpper || site.Value.Lower != 1 || site.Value.Upper != 1 {
			t.Fatalf("equality bounds: %+v", site.Value)
		}
	})
}

func reviewReport(t *testing.T, src string) runtimeguard.Report {
	t.Helper()
	graphs := runtimeTestGraphs(t, src)
	report, err := runtimeguard.Analyze(&graphs[len(graphs)-1].Pdg)
	if err != nil {
		t.Fatal(err)
	}
	return report
}
func reviewAccess(t *testing.T, src string) runtimeguard.Access {
	t.Helper()
	report := reviewReport(t, src)
	var indices []runtimeguard.Access
	for _, site := range report.Accesses {
		if site.Index {
			indices = append(indices, site)
		}
	}
	if len(indices) != 1 {
		t.Fatalf("got %d index accesses,want 1", len(indices))
	}
	return indices[0]
}

func TestRuntimePointerProofInvalidation(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"copy overwritten", `r:=p; r=q; if r==nil{return 0};return *p`, 1},
		{"source overwritten", `r:=p; p=nil; if r==nil{return 0};return *p`, 1},
		{"copy chain", `r:=p;s:=r;if s==nil{return 0};return *p`, 0},
		{"branch copies differ", `r:=p;if choose{r=q};if r==nil{return 0};return *p`, 1},
		{"reverse copy guard", `r:=p;if p==nil{return 0};return *r`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `package p;func f(p,q *int,choose bool)int{if q==nil{return 0};` + tc.body + `}`
			if got := unprovenPointerAccesses(t, src); len(got) != tc.want {
				t.Fatalf("got %d findings,want %d", len(got), tc.want)
			}
		})
	}
}

func TestRuntimeFieldProofInvalidation(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"write", `if p.Inner==nil{return 0};p.Inner=nil;return p.Value`, 1},
		{"unknown call", `if p.Inner==nil{return 0};mutate(p);return p.Value`, 1},
		{"copied field after mutation", `r:=p.Inner;mutate(p);if r==nil{return 0};return p.Value`, 1},
		{"nested value receiver", `if p.Inner==nil{return 0};return p.Value`, 0},
		{"field guard conditional", `if choose{if p.Inner==nil{return 0}};return p.Value`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `package p;type Inner struct{Value int};type Middle struct{*Inner};type Outer struct{Middle};func mutate(p *Outer){p.Inner=nil};func f(p *Outer,q *int,choose bool)int{if p==nil || q==nil{return 0};` + tc.body + `}`
			if got := unprovenPointerAccesses(t, src); len(got) != tc.want {
				t.Fatalf("got %d findings,want %d", len(got), tc.want)
			}
		})
	}
}

func TestRuntimeConcurrentFieldProofStaysUnknown(t *testing.T) {
	src := `package p;type Inner struct{Value int};type Outer struct{*Inner};func mutate(p *Outer){p.Inner=nil};func f(p *Outer,q *int)int{if p==nil || q==nil{return 0};go mutate(p);if p.Inner==nil{return 0};return p.Value}`
	if got := unprovenPointerAccesses(t, src); len(got) != 1 {
		t.Fatalf("concurrent field: got %d findings,want 1", len(got))
	}
}

func TestRuntimeFieldGuardIndependentOperations(t *testing.T) {
	for _, body := range []string{`x:=1;return p.Value+x`, `x:=len(s);return p.Value+x`, `x:=cap(s);return p.Value+x`} {
		src := `package p;type Inner struct{Value int};type Outer struct{*Inner};func f(p Outer,q *int,s []int)int{if q==nil || p.Inner==nil{return 0};` + body + `}`
		if got := unprovenPointerAccesses(t, src); len(got) != 0 {
			t.Fatalf("%s: %d false findings", body, len(got))
		}
	}
}

func TestRuntimePointerArrayNilGuard(t *testing.T) {
	for _, body := range []string{`if q==nil{return 0};return s[0]`, `if q==nil{return 0};s[0]=1;return 0`} {
		src := `package p;func f(s *[3]int,q *int)int{` + body + `}`
		if got := unprovenPointerAccesses(t, src); len(got) != 1 {
			t.Fatalf("%s: got %d findings,want 1", body, len(got))
		}
	}
	src := `package p;func f(s *[3]int,q *int)int{if s==nil || q==nil{return 0};return s[0]}`
	if got := unprovenPointerAccesses(t, src); len(got) != 0 {
		t.Fatalf("safe pointer array: %d findings", len(got))
	}
}

func TestRuntimeIntegerLimitReachability(t *testing.T) {
	for _, condition := range []string{`i < -9223372036854775808`, `i > 9223372036854775807`} {
		src := `package p;func f(i int64)int{if ` + condition + `{return []int{1}[int(i)]};return 0}`
		if report := reviewReport(t, src); len(report.Accesses) != 0 {
			t.Fatalf("%s: got %d unreachable accesses", condition, len(report.Accesses))
		}
	}
	src := `package p;func f(i uint64)int{if i > 9223372036854775807{return []int{1}[int(i)]};return 0}`
	if report := reviewReport(t, src); len(report.Accesses) != 1 {
		t.Fatalf("unsigned branch is reachable: %+v", report)
	}
}
