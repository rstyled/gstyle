package gopatterns

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
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
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	extraction, err := Extract(context.Background(), "", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	var sites []patterns.Site
	for _, fp := range extraction.Funcs {
		if fp.Name != "example.First" && fp.Name != "example.Second" {
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
	out, err := ApplyFix(spec, []byte(proofFixture))
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
	if _, err := ApplyFix(spec, append([]byte(proofFixture), '\n')); err == nil {
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
