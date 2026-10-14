package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/patterns"
	"os"
	"path/filepath"
	"testing"
)

func applyTypedFix(t *testing.T, spec *patterns.FixSpec, source []byte) ([]byte, error) {
	t.Helper()
	pkg := loadTestPackage(t, string(source))
	root := filepath.Dir(pkg.GoFiles[0])
	ext, err := (&FixExtractor{}).Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	spec.Context = ext.Sources
	spec.File = pkg.GoFiles[0]
	return ApplyFix(spec, source)
}

func TestSemanticFixRuntime(t *testing.T) {
	cases := []struct {
		kind             patterns.CandidateKind
		source, behavior string
		line             int
		params           map[string]string
	}{
		{patterns.GuardClause, "package probe\nfunc choose(x float64) int {\n value:=0\n if x>0 {value=1;value++} else{return -1}\n return value\n}\n", "package probe\nimport(\"testing\";\"math\")\nfunc TestPreserveNaN(t *testing.T){if got:=choose(math.NaN());got!=-1{t.Fatalf(\"NaN changed to %d\",got)}}\n", 4, map[string]string{"if_line": "4"}},
		{patterns.TypeSwitch, "package probe\nvar calls int\ntype A struct{}\ntype B struct{}\ntype C struct{}\nfunc (A) Speak(){calls++}\nfunc (B) Speak(){calls++}\nfunc (C) Speak(){calls++}\nfunc visit(x any){switch v:=x.(type){case A:v.Speak();case B:v.Speak()}}\n", "package probe\nimport \"testing\"\nfunc TestUnlistedType(t *testing.T){calls=0;visit(C{});if calls!=0{t.Fatalf(\"unlisted type called %d times\",calls)}}\n", 9, map[string]string{"bound": "v", "expr": "x", "method": "Speak", "types": "A,B", "args": ""}},
		{patterns.PrimitiveObsession, "package probe\nfunc first(email,message string)any{return message}\nfunc second(email string)bool{return email!=\"\"}\n", "package probe\nimport \"testing\"\nfunc TestMessageType(t *testing.T){if _,ok:=first(\"a\",\"b\").(string);!ok{t.Fatal(\"unrelated message parameter changed dynamic type\")}}\n", 2, nil},
		{patterns.EnumDispatch, "package probe\nimport \"net/http\"\nvar hits int\nfunc yes(){hits++}\nfunc no(){hits--}\nfunc visit(code int){switch code{case http.StatusOK:yes();case http.StatusNotFound:no()}}\n", "package probe\nimport \"testing\"\nfunc TestDispatch(t *testing.T){hits=0;visit(200);if hits!=1{t.Fatal(hits)}}\n", 6, nil},
		{patterns.Specification, "package probe\n\ntype Gate struct{ Ready, Allowed bool }\nfunc both(g Gate) int { if g.Ready && g.Allowed {return 1};return 0 }\nfunc either(g Gate) int { if g.Ready || g.Allowed {return 2};return 0 }\n\nfunc bothAgain(g Gate)int{if g.Ready&&g.Allowed{return 1};return 0}\n", "package probe\nimport \"testing\"\nfunc TestDifferentLogic(t *testing.T){g:=Gate{Ready:true};if both(g)!=0 || either(g)!=2 {t.Fatalf(\"boolean logic changed: both=%d either=%d\",both(g),either(g))}}\n", 3, map[string]string{"type": "Gate", "varname": "g", "cond": "g.Ready&&g.Allowed", "rulekey": "Gate|(Ready:truthy:&&Allowed:truthy:)"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			root, ext := sourceFixModule(t, map[string]string{"probe.go": tc.source, "probe_test.go": tc.behavior})
			runCoverageBehavior(t, root)
			path := filepath.Join(root, "probe.go")
			spec := &patterns.FixSpec{Kind: tc.kind, Context: ext.Sources, File: path, Line: tc.line, Params: tc.params}
			out, err := ApplyFix(spec, []byte(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if string(out) == tc.source {
				t.Fatal("fix was not exercised")
			}
			if err := os.WriteFile(path, out, 0600); err != nil {
				t.Fatal(err)
			}
			runCoverageBehavior(t, root)
		})
	}
}
