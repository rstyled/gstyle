package patternfix

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/application/patterncheck"
	"github.com/rstyled/gstyle/domain/patterns"
)

func TestOffsetFallbackAdvisoryShowsDiffWithoutApplying(t *testing.T) {
	src := `package example
func page(items []int, offset int) []int {
 if offset > 0 && offset <= len(items) { items = items[offset:] }
 return items
}
`
	path := writeTempGo(t, src)
	var report bytes.Buffer
	if err := patterncheck.Run(context.Background(), []string{path}, &report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"offset_fallback", "- if offset > 0 &&", "min(offset, len(items))", "proposal only"} {
		if !strings.Contains(report.String(), want) {
			t.Errorf("report missing %q:\n%s", want, report.String())
		}
	}
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--kind", "offset_fallback", "--apply", path}, &output); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != src {
		t.Fatal("advisory proposal was applied")
	}
}

func TestOffsetFallbackFixPreservesRuntimeResults(t *testing.T) {
	src := `package example
func page(items []int, offset int) []int {
 if offset > 0 && offset <= len(items) { items = items[offset:] } else { items = items[len(items):] }
 return items
}
`
	path := writeTempGo(t, src)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "go.mod"), []byte("module example\n\ngo 1.26\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Dir(path))
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--kind", "offset_fallback", "--apply", path}, &output); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "min(offset, len(items))") {
		t.Fatalf("fix was not applied:\n%s", content)
	}
	original := strings.Replace(src, "func page(", "func original(", 1)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "original.go"), []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	test := `package example
import ("reflect"; "testing")
func TestEquivalent(t *testing.T) {
 for _, input := range [][]int{nil, {}, {1,2}, make([]int,2,4)} {
  for offset := -1; offset <= 5; offset++ {
   before, after := original(input,offset), page(input,offset)
   if !reflect.DeepEqual(before,after) || cap(before)!=cap(after) {
    t.Errorf("offset %d: before=%v cap=%d, after=%v cap=%d",offset,before,cap(before),after,cap(after))
   }
  }
 }
}
`
	testPath := filepath.Join(filepath.Dir(path), "page_test.go")
	if err := os.WriteFile(testPath, []byte(test), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", path, filepath.Join(filepath.Dir(path), "original.go"), testPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixed code changed runtime results: %v\n%s", err, out)
	}
}

func TestOffsetFallbackKindIsAvailableInPhases(t *testing.T) {
	if err := (options{kind: "offset_fallback"}).validate(); err != nil {
		t.Fatal(err)
	}
	if !phaseHas(fixPhases[phaseIndex(patterns.OffsetFallback)], patterns.OffsetFallback) {
		t.Fatal("range guard fix missing from phased workflow")
	}
}
