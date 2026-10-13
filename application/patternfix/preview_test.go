package patternfix

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultPreviewMatchesApplyAcrossDependentPhases(t *testing.T) {
	const source = `package example

type Service struct{}
func helper(x int) int {
 if x > 0 {
  println(x)
  println("positive")
 } else { return -1 }
 return x
}
func (s *Service) Run() {}
func NewService() *Service { return &Service{} }
`
	path := writeTempGo(t, source)
	var preview, applied bytes.Buffer
	if err := Run(context.Background(), []string{path}, &preview); err != nil {
		t.Fatal(err)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != source {
		t.Fatal("preview writes source")
	}
	if err := Run(context.Background(), []string{"--apply", path}, &applied); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := unifiedDiff(path, []byte(source), result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.String(), expected) {
		t.Fatalf("preview differs from apply:\n%s\nexpected:\n%s", &preview, expected)
	}
	for _, kind := range []string{"guard_clause", "declaration_order"} {
		if !strings.Contains(preview.String(), "("+kind+")") {
			t.Fatalf("missing phase %s: %s", kind, &preview)
		}
	}
}

func TestPhasedOptionIsRemoved(t *testing.T) {
	if _, err := parseOptions([]string{"--phased"}); err == nil {
		t.Fatal("obsolete phased option still accepted")
	}
}

func TestChangedPreviewMatchesApplyAfterLineShifts(t *testing.T) {
	dir := changedRepo(t, changedClean)
	defer chdir(t, dir)()
	dirty := strings.Replace(changedClean, "func touched(x int) int {", "func touched(x int) int {\n\t// touched", 1)
	writeRepoFile(t, dir, "main.go", dirty)
	var preview, applied bytes.Buffer
	if err := Run(context.Background(), []string{"--changed", "."}, &preview); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), []string{"--changed", "--apply", "."}, &applied); err != nil {
		t.Fatal(err)
	}
	path, err := filepath.Abs("main.go")
	if err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := unifiedDiff(path, []byte(dirty), result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.String(), expected) {
		t.Fatalf("changed preview differs from apply: %s", &preview)
	}
	if !strings.Contains(string(result), "func untouched(x int) int {\n\tif x > 0") {
		t.Fatal("untouched function rewritten")
	}
}
