package gopatterns

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func extractedMethodSpec(t *testing.T, root string, extraction *Extraction) *patterns.FixSpec {
	t.Helper()
	candidates := patterns.RunFixCandidates(nil, patterns.Options{AnemicModels: extraction.AnemicModels}, patterns.AnemicModel)
	if len(candidates) != 1 {
		t.Fatalf("expected one typed method proposal, got %d", len(candidates))
	}
	spec := candidates[0].FixSpec
	spec.File = filepath.Join(root, spec.File)
	spec.Context = extraction.Sources
	return spec
}

func TestMethodTargetsAcrossFilesKeepValueAndPointerSemantics(t *testing.T) {
	root, extraction := sourceFixModule(t, map[string]string{
		"model.go": `package sample
// Item is a shared value.
type Item struct { Value int }
`,
		"operations.go": `package sample
// read keeps its comment.
func read(item Item) int { return item.Value }
func increment(item *Item) { item.Value++ }
func twice(item Item) int { return read(item)+item.Value }
func Exported(item Item) int { return item.Value }
func unrelated(item Item) int { return 0 }
`,
		"caller_test.go": `package sample
import "testing"
func TestItem(t *testing.T) { item:=Item{Value:2};increment(&item);if read(item)!=3 || twice(item)!=6 || Exported(item)!=3 {t.Fatal(item)} }
`,
	})
	spec := extractedMethodSpec(t, root, extraction)
	overlay, err := PlanSignatureFix(spec)
	if err != nil {
		t.Fatal(err)
	}
	operations := string(overlay[filepath.Join(root, "operations.go")])
	for _, want := range []string{"func (item Item) read()", "func (item *Item) increment()", "return item.read() + item.Value", "// read keeps its comment.", "func Exported(item Item)", "func unrelated(item Item)"} {
		if !strings.Contains(operations, want) {
			t.Fatalf("missing %s:\n%s", want, operations)
		}
	}
	caller := string(overlay[filepath.Join(root, "caller_test.go")])
	if !strings.Contains(caller, "item.read()") || !strings.Contains(caller, "(&item).increment()") {
		t.Fatalf("test callers not updated:\n%s", caller)
	}
	if _, ok := overlay[filepath.Join(root, "model.go")]; ok {
		t.Fatal("unchanged type declaration rewritten")
	}
	if err := ValidateFixOverlay(context.Background(), extraction.Sources, overlay); err != nil {
		t.Fatal(err)
	}
	next, err := ExtractFixOverlay(context.Background(), root, []string{"."}, SourceOverlay{Files: overlay, Disk: extraction.Sources})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.AnemicModels) != 0 {
		t.Fatal("completed method proposal repeats")
	}
}

func TestMethodTargetsRejectFunctionValuesAcrossFiles(t *testing.T) {
	root, extraction := sourceFixModule(t, map[string]string{
		"model.go":      "package sample\ntype Item struct{Value int}",
		"operations.go": "package sample\nfunc read(item Item)int{return item.Value}\nfunc twice(item Item)int{return item.Value*2}\nfunc thrice(item Item)int{return item.Value*3}",
		"caller.go":     "package sample\nvar callback=read",
	})
	if _, err := PlanSignatureFix(extractedMethodSpec(t, root, extraction)); err == nil {
		t.Fatal("function value bypasses reference checks")
	}
}

func TestAnemicDetectionUsesBindingsAndLocalTypeIdentity(t *testing.T) {
	_, extraction := sourceFixModule(t, map[string]string{
		"foreign/model.go": "package foreign\ntype Item struct{Value int}",
		"model.go":         "package sample\ntype Item struct{Value int}",
		"operations.go": `package sample
import "example.com/sourcefix/foreign"
func shadowA(item Item)int{ {item:=foreign.Item{Value:1};return item.Value} }
func shadowB(item Item)int{ {item:=foreign.Item{Value:2};return item.Value} }
func shadowC(item Item)int{ {item:=foreign.Item{Value:3};return item.Value} }
func foreignA(item foreign.Item)int{return item.Value}
func foreignB(item foreign.Item)int{return item.Value}
func foreignC(item foreign.Item)int{return item.Value}
`,
	})
	if len(extraction.AnemicModels) != 0 {
		t.Fatal("shadowed or imported bindings count as local behavior")
	}
}
