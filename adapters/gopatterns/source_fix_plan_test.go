package gopatterns

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func sourceFixModule(t *testing.T, files map[string]string) (string, *Extraction) {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.com/sourcefix\n\ngo 1.24\n"
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	extraction, err := ExtractForFix(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	return root, extraction
}

const valueSource = `package sample
// combine retains its doc comment.
func combine(left, right string) string {
 // both bindings come from extraction.
 return left + right
}
func reverse(left, right string) string { return right + left }
func decorate(left, right string) string { return combine(left, right) }
func ignored(_ string) string { return "ok" }
`

func valueSpec(root string, extraction *Extraction) *patterns.FixSpec {
	return &patterns.FixSpec{Kind: patterns.ValueObject, File: filepath.Join(root, "values.go"), Context: extraction.Sources, Params: map[string]string{"group": "left:string,right:string"}}
}

func TestSourceSignaturePlanUpdatesTypedCrossFileAndTestCalls(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{
		"values.go": valueSource,
		"callers.go": `package sample
func use() string { return decorate(combine("a", "b"), reverse("c", "d")) }
func shadow() string { combine := func(a,b string) string { return a+b }; return combine("x","y") }
`,
		"values_test.go": `package sample
import "testing"
func TestCall(t *testing.T) { if combine("a", "b") != "ab" { t.Fatal("wrong") } }
`,
	})
	overlay, err := PlanSignatureFix(valueSpec(root, ext))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateFixOverlay(context.Background(), ext.Sources, overlay); err != nil {
		t.Fatal(err)
	}
	callers := string(overlay[filepath.Join(root, "callers.go")])
	if !strings.Contains(callers, `combine("x", "y")`) {
		t.Fatal("shadowed callable changed")
	}
	if !strings.Contains(callers, `decorate(LeftRight{Left: combine(LeftRight{`) {
		t.Fatalf("nested calls not rewritten: %s", callers)
	}
	if !strings.Contains(string(overlay[filepath.Join(root, "values_test.go")]), "combine(LeftRight{") {
		t.Fatal("test call is missing")
	}
	values := string(overlay[filepath.Join(root, "values.go")])
	if !strings.Contains(values, "// combine retains its doc comment.") || !strings.Contains(values, "// both bindings come from extraction.") {
		t.Fatal("comments lost")
	}
	for _, fn := range ext.Funcs {
		if err := pdg.Validate(&fn.Pdg); err != nil {
			t.Fatal(err)
		}
		if fn.Pdg.Tables.Sources != ext.Sources || fn.Pdg.Sources[0].Snapshot == 0 {
			t.Fatal("graph does not share source snapshot")
		}
	}
}

func TestSourceSignaturePlanRejectsFunctionValues(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": valueSource, "other.go": "package sample\nvar saved = combine\n"})
	if _, err := PlanSignatureFix(valueSpec(root, ext)); err == nil {
		t.Fatal("function value accepted")
	}
}

func TestSourceSignaturePlanPreservesArgumentOrder(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": `package sample
func combine(right, left string) string { return right+left }
func reverse(right, left string) string { return left+right }
func decorate(right, left string) string { return combine(right,left) }
`, "other.go": "package sample\nfunc use() string { return combine(first(),second()) }\nfunc first() string { return \"a\" };func second() string{return \"b\"}\n"})
	overlay, err := PlanSignatureFix(valueSpec(root, ext))
	if err != nil {
		t.Fatal(err)
	}
	caller := string(overlay[filepath.Join(root, "other.go")])
	if !strings.Contains(caller, "Right: first(), Left: second()") {
		t.Fatal("evaluation order changed")
	}
	if err := ValidateFixOverlay(context.Background(), ext.Sources, overlay); err != nil {
		t.Fatal(err)
	}
}

func TestSourceMethodPlanHandlesCrossFileNilReceiver(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{
		"model.go": `package sample
 type Item struct { value int }
 // read retains its doc.
 func read(item *Item) int { if item == nil { return 0 }; return item.value }
 func valid(item *Item) bool { return item != nil }
 func next(item *Item) int { return read(item)+1 }
`,
		"callers.go": "package sample\nfunc use() int { return read(nil)+next(&Item{value:1}) }\n",
	})
	spec := &patterns.FixSpec{Kind: patterns.AnemicModel, File: filepath.Join(root, "model.go"), Context: ext.Sources, Params: map[string]string{"type_name": "Item"}}
	overlay, err := PlanSignatureFix(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateFixOverlay(context.Background(), ext.Sources, overlay); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(overlay[filepath.Join(root, "callers.go")]), "(*Item)(nil).read()") {
		t.Fatal("typed nil receiver not preserved")
	}
}

func TestFixValidationRejectsStaleSourceAndLostComments(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": valueSource})
	path := filepath.Join(root, "values.go")
	withoutComment := strings.Replace(valueSource, "// combine retains its doc comment.\n", "", 1)
	if err := ValidateFixOverlay(context.Background(), ext.Sources, map[string][]byte{path: []byte(withoutComment)}); err == nil {
		t.Fatal("lost comment accepted")
	}
	if err := os.WriteFile(path, []byte(valueSource+"\n// changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFixOverlay(context.Background(), ext.Sources, map[string][]byte{path: []byte(valueSource)}); err == nil {
		t.Fatal("stale source accepted")
	}
}

func TestSourceCommentRangesPreserveCRLFAndIgnoreLiterals(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": "package sample\r\n// doc\r\nvar text = `// literal /* literal */`\r\n/* block\r\ncomment */\r\n"})
	path := filepath.Join(root, "values.go")
	snapshot, ok := ext.Sources.Snapshot(ext.Sources.Reference(path))
	if !ok || len(snapshot.Comments) != 2 {
		t.Fatal("incorrect parser comments")
	}
	if err := ValidateFixOverlay(context.Background(), ext.Sources, map[string][]byte{path: []byte(snapshot.Content)}); err != nil {
		t.Fatal(err)
	}
}

func TestFixValidationRejectsMovedDeclarationComments(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": valueSource})
	moved := strings.Replace(valueSource, "// combine retains its doc comment.\n", "", 1)
	moved = strings.Replace(moved, "func reverse", "// combine retains its doc comment.\nfunc reverse", 1)
	err := ValidateFixOverlay(context.Background(), ext.Sources, map[string][]byte{filepath.Join(root, "values.go"): []byte(moved)})
	if err == nil || !strings.Contains(err.Error(), "declaration comment") {
		t.Fatalf("moved doc accepted: %v", err)
	}
}

func TestSourceSignaturePlanRejectsExcludedSource(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": valueSource, "excluded.go": "//go:build never_enabled\n\npackage sample\nvar hidden = combine\n"})
	_, err := PlanSignatureFix(valueSpec(root, ext))
	if err == nil || !strings.Contains(err.Error(), "coverage is incomplete") {
		t.Fatalf("excluded references accepted: %v", err)
	}
}

func TestSourceSignaturePlanRejectsAddressedParameters(t *testing.T) {
	source := strings.Replace(valueSource, "return left + right", "pointer := &left; return *pointer + right", 1)
	root, ext := sourceFixModule(t, map[string]string{"values.go": source})
	if _, err := PlanSignatureFix(valueSpec(root, ext)); err == nil {
		t.Fatal("parameter address accepted")
	}
}

func TestFixValidationRejectsChangedModuleConfiguration(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": valueSource})
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/changed\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckFixSources(ext.Sources); err == nil {
		t.Fatal("changed module configuration accepted")
	}
}

func TestFixValidationRejectsNewWorkspaceConfiguration(t *testing.T) {
	root, ext := sourceFixModule(t, map[string]string{"values.go": valueSource})
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.24\nuse .\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckFixSources(ext.Sources); err == nil {
		t.Fatal("new workspace configuration accepted")
	}
}
