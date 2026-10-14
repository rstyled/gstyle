package gopatterns

import (
	"context"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func coverageReport(extraction *Extraction, names ...string) patterns.PatternsReport {
	var sites []patterns.Site
	for _, name := range names {
		for _, function := range extraction.Funcs {
			if strings.HasSuffix(function.Name, "."+name) {
				sites = append(sites, patterns.Site{ID: function.Name, Name: function.Name, Path: function.Path, Line: function.Line, EndLine: function.EndLine})
			}
		}
	}
	return patterns.PatternsReport{Candidates: []patterns.Candidate{{Kind: patterns.Parameterize, Sites: sites}}}
}

func approvedCoverageSpecs(t *testing.T, extraction *Extraction, report *patterns.PatternsReport) []*patterns.FixSpec {
	t.Helper()
	if err := VerifyParameterizations(report, extraction); err != nil {
		t.Fatal(err)
	}
	var specs []*patterns.FixSpec
	for _, candidate := range report.Candidates {
		if candidate.FixSpec != nil {
			candidate.FixSpec.Context = extraction.Sources
			specs = append(specs, candidate.FixSpec)
		}
	}
	return specs
}

func TestParameterizationKeepsInterveningSourceAndComments(t *testing.T) {
	separator := "\n// Preserve this type and its literal.\ntype Marker struct{Note string}\nconst marker = `// literal text`\n"
	source := strings.Replace(proofFixture, "func Second", separator+"func Second", 1)
	source = strings.Replace(source, "func First", "// First keeps its docs.\nfunc First", 1)
	source = strings.Replace(source, "func Second", "// Second keeps its docs.\nfunc Second", 1)
	root, extraction := sourceFixModule(t, map[string]string{"sample.go": source})
	report := coverageReport(extraction, "First", "Second")
	specs := approvedCoverageSpecs(t, extraction, &report)
	if len(specs) != 1 {
		t.Fatalf("valid separated pair rejected: %+v", report.Candidates)
	}
	out, err := ApplyFix(specs[0], []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"// Preserve this type and its literal.", "type Marker struct", "const marker = `// literal text`", "// First keeps its docs.", "// Second keeps its docs."} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("lost %s:\n%s", want, out)
		}
	}
	if err := ValidateFixOverlay(context.Background(), extraction.Sources, map[string][]byte{filepath.Join(root, "sample.go"): out}); err != nil {
		t.Fatal(err)
	}
}

func TestParameterizationGroupsKeepAdvisoryAndSelectDisjointPairs(t *testing.T) {
	source, err := format.Source([]byte(proofFixture))
	if err != nil {
		t.Fatal(err)
	}
	root, extraction := sourceFixModule(t, map[string]string{"sample.go": string(source)})
	report := coverageReport(extraction, "First", "Second", "Third", "Fourth", "left")
	report.Candidates = append(report.Candidates, report.Candidates[0])
	specs := approvedCoverageSpecs(t, extraction, &report)
	if len(specs) != 2 {
		t.Fatalf("expected two unique proved pairs, got %d", len(specs))
	}
	if report.Candidates[0].FixSpec != nil || len(report.Candidates[0].CounterEvidence) == 0 {
		t.Fatal("whole group advertised as proved")
	}
	if specs[0].Params["helper_name"] == specs[1].Params["helper_name"] {
		t.Fatal("independent helpers collide")
	}
	snapshot, _ := extraction.Sources.Snapshot(extraction.Sources.Reference(filepath.Join(root, "sample.go")))
	out, count, _, err := PlanLocalFixes(specs, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("independent selected pairs overlap: %d", count)
	}
	if err := ValidateFixOverlay(context.Background(), extraction.Sources, map[string][]byte{snapshot.Path: out}); err != nil {
		t.Fatal(err)
	}
}

func TestCrossFileParameterizationPreservesImportsCallsAndComments(t *testing.T) {
	cases := []struct{ name, imports, extra string }{
		{"single", `import "strings"`, ""},
		{"grouped", `import ("strings")`, ""},
		{"partial group", `import ("strings";"errors")`, `var kept = errors.New("keep")`},
		{"import docs", "// Keep the import note.\nimport \"strings\"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := `package sample
import "strings"
// First keeps its docs.
func First(x int)(string,error){
 // first body note
 value,err:=left(x)
 if err!=nil{return "",err}
 return strings.TrimSpace(value),nil
}`
			second := `package sample
` + tc.imports + `
// Second keeps its docs.
func Second(x int)(string,error){
 // second body note
 value,err:=right(x)
 if err!=nil{return "",err}
 return strings.TrimSpace(value),nil
}
` + tc.extra
			root, extraction := sourceFixModule(t, map[string]string{"first.go": first, "second.go": second, "calls.go": coverageCalls, "behavior_test.go": coverageBehavior})
			runCoverageBehavior(t, root)
			report := coverageReport(extraction, "First", "Second")
			specs := approvedCoverageSpecs(t, extraction, &report)
			if len(specs) != 1 {
				t.Fatalf("cross-file proof rejected: %+v", report.Candidates)
			}
			overlay, err := PlanParameterizationFix(specs[0])
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateFixOverlay(context.Background(), extraction.Sources, overlay); err != nil {
				t.Fatal(err)
			}
			b := string(overlay[filepath.Join(root, "second.go")])
			if !strings.Contains(b, "// Second keeps its docs.") || !strings.Contains(b, "// second body note") {
				t.Fatalf("lost secondary comments:\n%s", b)
			}
			if tc.extra != "" && !strings.Contains(b, tc.extra) {
				t.Fatalf("unrelated import user changed:\n%s", b)
			}
			for path, content := range overlay {
				if err := os.WriteFile(path, content, 0644); err != nil {
					t.Fatal(err)
				}
			}
			runCoverageBehavior(t, root)
		})
	}
}

const coverageCalls = `package sample
import "errors"
var events []string
func next()int{events=append(events,"argument");return 1}
func left(x int)(string,error){events=append(events,"left");if x<0{return "",errors.New("bad")};return " left ",nil}
func right(x int)(string,error){events=append(events,"right");if x<0{return "",errors.New("bad")};return " right ",nil}
`
const coverageBehavior = `package sample
import("testing";"reflect")
func TestBehavior(t *testing.T){
 events=nil
 got,err:=First(next())
 if got!="left" || err!=nil || !reflect.DeepEqual(events,[]string{"argument","left"}){t.Fatal(got,err,events)}
 events=nil
 got,err=Second(-1)
 if got!="" || err==nil || !reflect.DeepEqual(events,[]string{"right"}){t.Fatal(got,err,events)}
 callbacks:=[]func(int)(string,error){First,Second}
 for _,fn:=range callbacks {if _,err:=fn(1);err!=nil{t.Fatal(err)}}
}
`

func runCoverageBehavior(t *testing.T, root string) {
	t.Helper()
	command := exec.Command("go", "test", "-count=1", "./...")
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("behavior changed: %v\n%s", err, out)
	}
}

func TestCrossFileParameterizationUsesCanonicalTypesAndImportBindings(t *testing.T) {
	first := `package sample
import("time"; clean "path")
func First(x time.Time)(string,error){value,err:=left(x);if err!=nil{return "",err};return clean.Clean(value),nil}`
	second := `package sample
import(clock "time"; folder "path")
func Second(x clock.Time)(string,error){value,err:=right(x);if err!=nil{return "",err};return folder.Clean(value),nil}`
	calls := `package sample
import "time"
func left(time.Time)(string,error){return "a/../left",nil}
func right(time.Time)(string,error){return "b/../right",nil}`
	behavior := `package sample
import("testing";"time")
func TestValues(t *testing.T){a,e:=First(time.Time{});b,f:=Second(time.Time{});if a!="left" || b!="right" || e!=nil || f!=nil{t.Fatal(a,b,e,f)}}`
	root, extraction := sourceFixModule(t, map[string]string{"first.go": first, "second.go": second, "calls.go": calls, "behavior_test.go": behavior})
	report := coverageReport(extraction, "First", "Second")
	specs := approvedCoverageSpecs(t, extraction, &report)
	if len(specs) != 1 {
		t.Fatalf("canonical types rejected because of spelling: %+v", report.Candidates)
	}
	overlay, err := PlanParameterizationFix(specs[0])
	if err != nil {
		t.Fatal(err)
	}
	b := string(overlay[filepath.Join(root, "second.go")])
	if !strings.Contains(b, `clock "time"`) || !strings.Contains(b, "x clock.Time") || strings.Contains(b, `folder "path"`) {
		t.Fatalf("wrapper type spelling or imports changed incorrectly:\n%s", b)
	}
	if err := ValidateFixOverlay(context.Background(), extraction.Sources, overlay); err != nil {
		t.Fatal(err)
	}
	for path, content := range overlay {
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	runCoverageBehavior(t, root)
}

func TestCrossFileParameterizationRejectsDifferentImportsAndStaleSources(t *testing.T) {
	first := `package sample
import codec "path"
func First(x int)(string,error){value,err:=left(x);if err!=nil{return "",err};return codec.Clean(value),nil}`
	second := strings.ReplaceAll(strings.ReplaceAll(first, "First", "Second"), "left(x)", "right(x)")
	t.Run("different package binding", func(t *testing.T) {
		_, extraction := sourceFixModule(t, map[string]string{"first.go": first, "second.go": strings.Replace(second, `"path"`, `"path/filepath"`, 1), "calls.go": coverageCalls})
		report := coverageReport(extraction, "First", "Second")
		if specs := approvedCoverageSpecs(t, extraction, &report); len(specs) != 0 {
			t.Fatal("same spelling hid different imported operations")
		}
		if !strings.Contains(strings.Join(report.Candidates[0].CounterEvidence, ";"), "different imported bindings") {
			t.Fatalf("missing import evidence: %+v", report.Candidates)
		}
	})
	t.Run("stale secondary", func(t *testing.T) {
		root, extraction := sourceFixModule(t, map[string]string{"first.go": first, "second.go": second, "calls.go": coverageCalls})
		report := coverageReport(extraction, "First", "Second")
		specs := approvedCoverageSpecs(t, extraction, &report)
		if len(specs) != 1 {
			t.Fatalf("missing proof: %+v", report.Candidates)
		}
		if err := os.WriteFile(filepath.Join(root, "second.go"), []byte(second+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := PlanParameterizationFix(specs[0]); err == nil {
			t.Fatal("secondary source changed without re-analysis")
		}
	})
}
