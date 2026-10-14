package patternfix

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestUnsupportedSignatureDoesNotBlockOtherFixes(t *testing.T) {
	path := writeTempGo(t, `package sample
func Exported(id int, name string) {}
func guard(x int) int {
 if x > 0 {
  println(x)
  println("positive")
 } else { return -1 }
 return x
}
`)
	ext, err := gopatterns.ExtractForFix(context.Background(), "", []string{path})
	if err != nil {
		t.Fatal(err)
	}
	bad := &patterns.FixSpec{Kind: patterns.ValueObject, File: path, Context: ext.Sources, Params: map[string]string{"group": "id:int,name:string"}}
	good := &patterns.FixSpec{Kind: patterns.GuardClause, File: path, Line: 4, Context: ext.Sources, Params: map[string]string{"if_line": "4"}}
	var output bytes.Buffer
	count, _, err := applyAndTrack(context.Background(), []*patterns.FixSpec{bad, good}, options{}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || !strings.Contains(output.String(), "skipped value_object") || !strings.Contains(output.String(), "if x <= 0") {
		t.Fatalf("unsupported candidate blocks valid fix: %d %s", count, &output)
	}
}

func TestSignatureOverlapDoesNotPartiallyAppendFiles(t *testing.T) {
	sources := pdg.NewSourceContext(t.TempDir(), nil)
	first, second := filepath.Join(sources.Root, "first.go"), filepath.Join(sources.Root, "second.go")
	sources.Add(first, []byte("first"), nil)
	sources.Add(second, []byte("second"), nil)
	plan := fixPlan{sources: sources, files: []filePlan{{path: first, after: []byte("accepted")}}}
	overlay := map[string][]byte{first: []byte("conflict"), second: []byte("partial")}
	if err := appendSignatureFiles(&plan, overlay, first); !errors.Is(err, errOverlappingPlans) {
		t.Fatalf("expected overlap: %v", err)
	}
	if len(plan.files) != 1 || string(plan.files[0].after) != "accepted" {
		t.Fatal("rejected plan left partial edits")
	}
}

func TestMissingSignatureEvidenceIsStillFatal(t *testing.T) {
	plan := fixPlan{}
	spec := &patterns.FixSpec{Kind: patterns.ValueObject}
	if err := addSignaturePlans(&plan, []*patterns.FixSpec{spec}); err == nil || len(plan.skipped) != 0 {
		t.Fatalf("missing evidence became a skip: %v", err)
	}
}

func TestUnknownSignatureTargetIsNotSilentlySkipped(t *testing.T) {
	sources := pdg.NewSourceContext(t.TempDir(), nil)
	path := filepath.Join(sources.Root, "code.go")
	sources.Add(path, []byte("package sample\n"), nil)
	plan := fixPlan{sources: sources}
	spec := &patterns.FixSpec{Kind: patterns.ValueObject, File: path, Context: sources, Targets: []string{"missing.target"}, Params: map[string]string{"group": "id:int,name:string"}}
	if err := addSignaturePlans(&plan, []*patterns.FixSpec{spec}); err == nil || len(plan.skipped) != 0 {
		t.Fatalf("unknown target became a skip: %v", err)
	}
}

func TestSignatureCountsOnceWhenTypeFileIsUnchanged(t *testing.T) {
	sources := pdg.NewSourceContext(t.TempDir(), nil)
	primary := filepath.Join(sources.Root, "model.go")
	first, second := filepath.Join(sources.Root, "a.go"), filepath.Join(sources.Root, "b.go")
	sources.Add(first, []byte("first"), nil)
	sources.Add(second, []byte("second"), nil)
	plan := fixPlan{sources: sources}
	if err := appendSignatureFiles(&plan, map[string][]byte{first: []byte("updated first"), second: []byte("updated second")}, primary); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, file := range plan.files {
		total += file.count
		if file.count > 0 && file.path != first {
			t.Fatal("count attribution is not deterministic")
		}
	}
	if total != 1 {
		t.Fatalf("accepted signature reported %d fixes", total)
	}
}

func TestPackageMethodPreviewApplyAndSecondRun(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":         "module example.com/methods\n\ngo 1.24\n",
		"model.go":       "package sample\n// Item stores a value.\ntype Item struct{Value int}\n",
		"operations.go":  "package sample\n// read keeps its comment.\nfunc read(item Item)int{return item.Value}\nfunc twice(item Item)int{return read(item)+item.Value}\nfunc thrice(item Item)int{return twice(item)+item.Value}\n",
		"caller_test.go": "package sample\nimport \"testing\"\nfunc TestItem(t *testing.T){if thrice(Item{Value:2})!=6{t.Fatal(\"bad value\")}}\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	defer chdir(t, root)()
	var preview, applied, repeat bytes.Buffer
	if err := Run(context.Background(), []string{"--kind", "all", "."}, &preview); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "operations.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != files["operations.go"] {
		t.Fatal("preview writes source")
	}
	if err := Run(context.Background(), []string{"--kind", "all", "--apply", "."}, &applied); err != nil {
		t.Fatal(err)
	}
	for name, original := range files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		path := filepath.Join(root, name)
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := unifiedDiff(path, []byte(original), after)
		if err != nil {
			t.Fatal(err)
		}
		if expected != "" && !strings.Contains(preview.String(), expected) {
			t.Fatalf("preview differs from apply for %s", name)
		}
	}
	if !strings.Contains(applied.String(), "phase 10 (anemic_model): fixed 1 candidate(s)") {
		t.Fatalf("method fix not counted: %s", &applied)
	}
	if err := Run(context.Background(), []string{"--kind", "all", "."}, &repeat); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(repeat.String(), "0 fixable candidate(s)") {
		t.Fatalf("second run still fixes source: %s", &repeat)
	}
}
