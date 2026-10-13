package gopatterns

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

const proofFixture = `package example

func First(x int) (int, error) {
 value, err := left(x)
 if err != nil { return 0, err }
 return len(value), nil
}
func Second(x int) (int, error) {
 value, err := right(x)
 if err != nil { return 0, err }
 return len(value), nil
}
func Third(x int) (int, error) {
 value, err := left(x)
 if err != nil { return 0, err }
 return len(value), nil
}
func Fourth(x int) (int, error) {
 value, err := right(x)
 if err != nil { return 0, err }
 return len(value), nil
}
func left(int) (string, error) { return "a", nil }
func right(int) (string, error) { return "bb", nil }
`

func proofExtraction(t *testing.T, src string) (*Extraction, patterns.PatternsReport) {
	return proofExtractionWithExtra(t, src, "")
}

func proofExtractionWithExtra(t *testing.T, src, extra string) (*Extraction, patterns.PatternsReport) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if extra != "" {
		if err := os.WriteFile(filepath.Join(dir, "extra.go"), []byte(extra), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	extraction, err := Extract(context.Background(), "", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	var sites []patterns.Site
	for _, fp := range extraction.Funcs {
		if !strings.HasPrefix(fp.Name, "example.First") && !strings.HasPrefix(fp.Name, "example.Second") {
			continue
		}
		sites = append(sites, patterns.Site{ID: fp.Name, Name: fp.Name, Path: fp.Path, Line: fp.Line, EndLine: fp.EndLine})
	}
	return extraction, patterns.PatternsReport{Candidates: []patterns.Candidate{{Kind: patterns.Parameterize, Sites: sites}}}
}

func TestPDGProofUsesCalleeResultsAndOnlySelectedPair(t *testing.T) {
	extraction, report := proofExtraction(t, proofFixture)
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	spec := report.Candidates[0].FixSpec
	if spec == nil {
		t.Fatalf("missing proof: %+v", report.Candidates[0])
	}
	out, err := applyProjectedFix(t, spec, []byte(proofFixture))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "fn func(int) (string, error)") {
		t.Fatalf("callee result inferred from enclosing int result: %s", out)
	}
	for _, name := range []string{"Third", "Fourth"} {
		if !strings.Contains(string(out), "func "+name+"(x int) (int, error) {\n\tvalue, err :=") {
			t.Fatalf("unselected pair changed: %s", out)
		}
	}
	if err := os.WriteFile("sample.go", out, 0644); err != nil {
		t.Fatal(err)
	}
	test := `package example
import "testing"
func TestResults(t *testing.T) {
 for i, fn := range []func(int)(int,error){First,Second,Third,Fourth} {
  got, err := fn(3)
  want := 1+i%2
  if err != nil || got != want { t.Fatalf("%d: got %d,%v want %d",i,got,err,want) }
 }
}`
	if err := os.WriteFile("sample_test.go", []byte(test), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./...")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code: %v\n%s", err, output)
	}
	if _, err := applyProjectedFix(t, spec, append([]byte(proofFixture), '\n')); err == nil {
		t.Fatal("stale source proof accepted")
	}
}

func TestPDGProofRejectsUnknownAndMismatchedCallTypes(t *testing.T) {
	cases := []struct {
		name, src string
		erase     bool
	}{
		{"unknown", proofFixture, true},
		{"incompatible", strings.ReplaceAll(proofFixture, "func right(int) (string, error) { return \"bb\", nil }", "func right(int) ([]byte, error) { return []byte(\"bb\"), nil }"), false},
		{"extra operation", strings.Replace(proofFixture, "value, err := right(x)", "x++\n value, err := right(x)", 1), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			extraction, report := proofExtraction(t, c.src)
			if c.erase {
				for i := range extraction.Funcs {
					graph := &extraction.Funcs[i].Pdg
					for j := range graph.Attributes {
						graph.Attributes[j].Extensions = 0
					}
				}
			}
			if err := VerifyParameterizations(&report, extraction); err != nil {
				t.Fatal(err)
			}
			if report.Candidates[0].FixSpec != nil {
				t.Fatal("unsupported pair has an auto-fix")
			}
			if len(report.Candidates[0].CounterEvidence) == 0 {
				t.Fatal("missing reason")
			}
		})
	}
}

func TestPDGProofRejectsStaleGraphs(t *testing.T) {
	extraction, report := proofExtraction(t, proofFixture)
	if err := os.WriteFile("sample.go", []byte(proofFixture+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	candidate := report.Candidates[0]
	if candidate.FixSpec != nil || !strings.Contains(strings.Join(candidate.CounterEvidence, " "), "snapshot") {
		t.Fatalf("stale graphs accepted: %+v", candidate)
	}
}

func TestParameterizeFixRequiresProof(t *testing.T) {
	_, err := ApplyFix(&patterns.FixSpec{Kind: patterns.Parameterize, File: "sample.go"}, []byte(proofFixture))
	if err == nil {
		t.Fatal("unverified fix accepted")
	}
}

func TestCallableProofRequiresRenderableSourceType(t *testing.T) {
	cases := []struct {
		a, b callableProof
		want bool
	}{
		{callableProof{"func(int) string", "func(int) string"}, callableProof{"func(int) string", "func(int) string"}, true},
		{callableProof{"func(int) remote.Type", ""}, callableProof{"func(int) remote.Type", ""}, false},
		{callableProof{"func(int) A", "func(int) A"}, callableProof{"func(int) B", "func(int) B"}, false},
	}
	for _, c := range cases {
		if got := compatibleCallableProofs(c.a, c.b); got != c.want {
			t.Errorf("got %v, want %v for %+v", got, c.want, c)
		}
	}
}

func TestParameterizationPreservesOccurrencesBindingsAndEffectOrder(t *testing.T) {
	const source = `package example
var calls []string
// First keeps its API documentation.
func First(fn1 int) int {
 // Keep left as a function value.
 value := left
 return value(fn1) + left(fn1) + left(fn1)
}
// Second keeps its API documentation.
func Second(fn1 int) int {
 value := left
 return value(fn1) + right(fn1) + other(fn1)
}
func left(x int) int { calls = append(calls, "left"); return x+1 }
func right(x int) int { calls = append(calls, "right"); return x+10 }
func other(x int) int { calls = append(calls, "other"); return x+100 }
`
	extraction, report := proofExtractionWithExtra(t, source, "package example\ntype extracted int\n")
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	spec := report.Candidates[0].FixSpec
	if spec == nil {
		t.Fatalf("valid substitutions rejected: %+v", report.Candidates[0])
	}
	if spec.Params["helper_name"] == "extracted" {
		t.Fatal("helper collides with type in another file")
	}
	testSource := `package example
import("testing";"reflect")
func TestPreserved(t *testing.T) {
 calls=nil
 if First(4)!=15 || Second(4)!=123 { t.Fatal("wrong substituted call or binding") }
 if !reflect.DeepEqual(calls,[]string{"left","left","left","left","right","other"}) { t.Fatal(calls) }
}`
	if err := os.WriteFile("sample_test.go", []byte(testSource), 0644); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)
	out, err := applyProjectedFix(t, spec, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "// Keep left as a function value.") || !strings.Contains(string(out), "value := left") {
		t.Fatalf("unmatched occurrence changed: %s", out)
	}
	if !strings.Contains(string(out), "// First keeps its API documentation.\nfunc First") || !strings.Contains(string(out), "// Second keeps its API documentation.\nfunc Second") {
		t.Fatalf("wrapper documentation moved or lost: %s", out)
	}
	if err := os.WriteFile("sample.go", out, 0644); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)
}

func assertParameterizationModule(t *testing.T) {
	t.Helper()
	if out, err := exec.Command("go", "test", "./...").CombinedOutput(); err != nil {
		t.Fatalf("module: %v\n%s", err, out)
	}
}

func TestParameterizationVariadicWrapperForwardsSlice(t *testing.T) {
	const source = `package example
func First(xs ...int) int { return left(xs...) }
func Second(xs ...int) int { return right(xs...) }
func left(xs ...int) int { return len(xs) }
func right(xs ...int) int { return len(xs)+1 }
`
	extraction, report := proofExtraction(t, source)
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	spec := report.Candidates[0].FixSpec
	if spec == nil {
		t.Fatalf("variadic proof missing: %+v", report.Candidates[0])
	}
	out, err := applyProjectedFix(t, spec, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sample.go", out, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sample_test.go", []byte(`package example
import "testing"
func TestResults(t *testing.T){ if First(1,2)!=2 || Second(1,2)!=3 { t.Fatal("wrong slice forwarding") } }
`), 0644); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)
}

func TestParameterizationRejectsRewriteSpecificRisks(t *testing.T) {
	cases := []struct{ name, source string }{
		{"directives", `package example
//go:noinline
func First() int { return left() }
func Second() int { return right() }
func left() int {return 1}
func right() int {return 2}`},
		{"blank forwarding", `package example
func First(_ int) int {return left()}
func Second(_ int) int {return right()}
func left()int{return 1}
func right()int{return 2}`},
		{"unnamed forwarding", `package example
func First(int) int {return left()}
func Second(int) int {return right()}
func left()int{return 1}
func right()int{return 2}`},
		{"builtins", `package example
func First(xs []int) int { return len(xs) }
func Second(xs []int) int { return cap(xs) }`},
		{"recover", `package example
func First() any { value:=recover(); left(); return value }
func Second() any { value:=recover(); right(); return value }
func left(){}
func right(){}`},
		{"spread mismatch", `package example
func First(xs []any) int { return left(xs...) }
func Second(xs []any) int { return right(xs) }
func left(xs ...any) int{return len(xs)}
func right(xs ...any) int{return len(xs)}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			extraction, report := proofExtraction(t, c.source)
			if err := VerifyParameterizations(&report, extraction); err != nil {
				t.Fatal(err)
			}
			if report.Candidates[0].FixSpec != nil || len(report.Candidates[0].CounterEvidence) == 0 {
				t.Fatalf("unsafe rewrite accepted: %+v", report.Candidates[0])
			}
		})
	}
}

func TestParameterizationRejectsChangedBindingIdentity(t *testing.T) {
	extraction, report := proofExtraction(t, proofFixture)
	for i := range extraction.Funcs {
		if extraction.Funcs[i].Name == "example.Second" {
			graph := &extraction.Funcs[i].Pdg
			graph.Symbols[2].Scope = graph.Function.ID
		}
	}
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	if report.Candidates[0].FixSpec != nil || !strings.Contains(strings.Join(report.Candidates[0].CounterEvidence, " "), "binding") {
		t.Fatal(report.Candidates[0])
	}
}

func TestParameterizationVoidWrappersPreserveCalls(t *testing.T) {
	const source = `package example
var result int
func First(){ left() }
func Second(){ right() }
func left(){ result+=1 }
func right(){ result+=2 }
`
	extraction, report := proofExtraction(t, source)
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	spec := report.Candidates[0].FixSpec
	if spec == nil {
		t.Fatal(report.Candidates[0])
	}
	out, err := applyProjectedFix(t, spec, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sample.go", out, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sample_test.go", []byte(`package example
import "testing"
func TestVoid(t *testing.T){ First();Second();if result!=3{t.Fatal(result)} }
`), 0644); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)
}

func TestParameterizationRejectsInvalidGraphBeforeCallableLookup(t *testing.T) {
	extraction, report := proofExtraction(t, proofFixture)
	for i := range extraction.Funcs {
		if extraction.Funcs[i].Name != "example.First" {
			continue
		}
		graph := &extraction.Funcs[i].Pdg
		for j := range graph.Nodes {
			if graph.Nodes[j].Category == pdg.Call {
				graph.Nodes[j].Span = 9999
			}
		}
	}
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	if report.Candidates[0].FixSpec != nil || len(report.Candidates[0].CounterEvidence) == 0 {
		t.Fatal("invalid graph accepted")
	}
}

func TestHelperNameDoesNotShadowPredeclaredNames(t *testing.T) {
	const source = `package example
func FirstLen(x int)int{return left(x)}
func SecondLen(x int)int{return right(x)}
func left(x int)int{return x}
func right(x int)int{return x+1}
`
	extraction, report := proofExtractionWithExtra(t, source, "package example\nfunc BuiltinUse()int{return len([]int{1,2})}\n")
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	spec := report.Candidates[0].FixSpec
	if spec == nil {
		t.Fatal(report.Candidates[0])
	}
	if spec.Params["helper_name"] == "len" {
		t.Fatal("helper shadows builtin in another file")
	}
	out, err := applyProjectedFix(t, spec, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sample.go", out, 0644); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)
}
