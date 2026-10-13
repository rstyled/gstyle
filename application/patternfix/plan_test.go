package patternfix

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/adapters/gopatterns"
)

func TestFixBatchCompileFailureLeavesAllFilesUnchanged(t *testing.T) {
	root := t.TempDir()
	source := "package sample\nfunc first() int { return 1 }\n"
	caller := "package sample\nfunc second() int { return first() }\n"
	files := map[string]string{"go.mod": "module example.com/plan\n\ngo 1.24\n", "first.go": source, "second.go": caller}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	extraction, err := gopatterns.ExtractForFix(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	plan := fixPlan{sources: extraction.Sources, files: []filePlan{
		{path: filepath.Join(root, "first.go"), before: []byte(source), after: []byte("package sample\nfunc first(x int) int{return x}\n"), count: 1},
		{path: filepath.Join(root, "second.go"), before: []byte(caller), after: []byte("package sample\nfunc second() int{return first()}\n"), count: 1},
	}}
	var output bytes.Buffer
	if _, _, err := executeFixPlan(context.Background(), plan, options{apply: true}, &output); err == nil || !strings.Contains(err.Error(), "does not compile") {
		t.Fatalf("expected compile rejection, got %v", err)
	}
	for name, original := range files {
		current, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(current) != original {
			t.Fatalf("%s changed before validation finished", name)
		}
	}
	if output.Len() != 0 {
		t.Fatal("reported writes on failed validation")
	}
}

func TestFixBatchRetainsPermissions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.go")
	if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writePlanFile(filePlan{path: path, after: []byte("after")}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatal("permissions changed")
	}
}
