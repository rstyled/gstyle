package gopatterns

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

const rangeFlowSource = `package sample
func width(lower,upper int) int { return upper-lower }
func center(lower,upper int) int { return lower+(upper-lower)/2 }
func between(lower,upper, value int) bool { return lower<=value && value<=upper }
func workflow(start,end int) int { return width(start,end)+center(start,end) }
`

func rangeGroupSpec(root string, ext *Extraction) *patterns.FixSpec {
	return &patterns.FixSpec{Kind: patterns.ValueObject, File: filepath.Join(root, "values.go"), Context: ext.Sources, Params: map[string]string{"group": "lower:int,upper:int"}}
}

func TestSharedValueIsCreatedOnceAndPassedThrough(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": rangeFlowSource})
	overlay, err := PlanSignatureFix(rangeGroupSpec(root, ext))
	if err != nil {
		t.Fatal(err)
	}
	result := string(overlay[filepath.Join(root, "values.go")])
	for _, want := range []string{"lowerUpper := LowerUpper{Lower: start, Upper: end}", "width(lowerUpper) + center(lowerUpper)"} {
		if !strings.Contains(result, want) {
			t.Fatalf("missing shared instance %s:\n%s", want, result)
		}
	}
	if strings.Count(result, "LowerUpper{") != 1 {
		t.Fatalf("group reconstructed at every call:\n%s", result)
	}
	if err := ValidateFixOverlay(context.Background(), ext.Sources, overlay); err != nil {
		t.Fatal(err)
	}
	next, err := ExtractFixOverlay(context.Background(), root, []string{"."}, SourceOverlay{Files: overlay, Disk: ext.Sources})
	if err != nil {
		t.Fatal(err)
	}
	methods, err := PlanSignatureFix(&patterns.FixSpec{Kind: patterns.AnemicModel, File: filepath.Join(root, "values.go"), Context: next.Sources, Params: map[string]string{"type_name": "LowerUpper"}})
	if err != nil {
		t.Fatal(err)
	}
	final := string(methods[filepath.Join(root, "values.go")])
	if !strings.Contains(final, "lowerUpper.width() + lowerUpper.center()") {
		t.Fatalf("later phase loses shared instance:\n%s", final)
	}
	if err := ValidateFixOverlay(context.Background(), next.Sources, methods); err != nil {
		t.Fatal(err)
	}
}

func TestSharedValueRejectsWeakOrChangingBindings(t *testing.T) {
	cases := []struct{ name, workflow string }{
		{"single operation", "func workflow(start,end int) int { return width(start,end) }"},
		{"same operation twice", "func workflow(start,end int) int { return width(start,end)+width(start,end) }"},
		{"reassigned", "func workflow(start,end int) int { first:=width(start,end);start++;return first+center(start,end) }"},
		{"captured write", "func workflow(start,end int) int { mutate:=func(){start++};first:=width(start,end);mutate();return first+center(start,end) }"},
		{"shadowed", "func workflow(start,end int) int { first:=width(start,end);{ start:=3;return first+center(start,end) } }"},
		{"different inputs", "func workflow(start,end,other int) int { return width(start,end)+center(start,other) }"},
	}
	declarations := rangeFlowSource[:strings.Index(rangeFlowSource, "func workflow")]
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, ext := sourceFixModule(t, map[string]string{"values.go": declarations + tc.workflow})
			if _, err := PlanSignatureFix(rangeGroupSpec(root, ext)); err == nil {
				t.Fatal("unproved grouping accepted")
			}
		})
	}
}

func TestFormattingShapeStaysAdvisory(t *testing.T) {
	source := `package sample
func truncate(value string,width int) string { if width<len(value){return value[:width]};return value }
func padText(value string,width int) string { return value }
func styledCell(value string,width int) string { value=truncate(value,width);return padText(value,width) }
`
	root, ext := sourceFixModule(t, map[string]string{"values.go": source})
	spec := &patterns.FixSpec{Kind: patterns.ValueObject, File: filepath.Join(root, "values.go"), Context: ext.Sources, Params: map[string]string{"group": "value:string,width:int"}}
	if _, err := PlanSignatureFix(spec); err == nil {
		t.Fatal("formatting wrapper automatically created")
	}
	report := patterns.PatternsReport{Candidates: []patterns.Candidate{{Kind: patterns.ValueObject, FixSpec: spec}}}
	VerifyValueGroups(&report, ext)
	if report.Candidates[0].FixSpec != nil || len(report.Candidates[0].CounterEvidence) == 0 {
		t.Fatal("weak evidence still advertises automatic fix")
	}
}

func TestSharedValuesKeepDistinctTuplesAndAvoidBindings(t *testing.T) {
	declarations := rangeFlowSource[:strings.Index(rangeFlowSource, "func workflow")]
	source := declarations + `func workflow(start,end,other,finish int) int {
 lowerUpper := 1
 return lowerUpper+width(start,end)+center(start,end)+width(other,finish)+center(other,finish)
 }`
	root, ext := sourceFixModule(t, map[string]string{"values.go": source})
	overlay, err := PlanSignatureFix(rangeGroupSpec(root, ext))
	if err != nil {
		t.Fatal(err)
	}
	result := string(overlay[filepath.Join(root, "values.go")])
	if strings.Count(result, "LowerUpper{") != 2 {
		t.Fatalf("distinct input tuples lost:\n%s", result)
	}
	for _, want := range []string{"lowerUpper1 :=", "lowerUpper2 :=", "width(lowerUpper1) + center(lowerUpper1)", "width(lowerUpper2) + center(lowerUpper2)"} {
		if !strings.Contains(result, want) {
			t.Fatalf("missing %s:\n%s", want, result)
		}
	}
	if err := ValidateFixOverlay(context.Background(), ext.Sources, overlay); err != nil {
		t.Fatal(err)
	}
}
