package gopatterns

import (
	"context"
	"errors"
	"github.com/rstyled/gstyle/domain/patterns"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOverlayChecksChangedImportersAndTests(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":       "module example.com/overlay\n\ngo 1.24\n",
		"lib/lib.go":   "package lib\nfunc Read() int { return 1 }\n",
		"main.go":      "package overlay\nimport \"example.com/overlay/lib\"\nfunc Read() int { return lib.Read() }\n",
		"main_test.go": "package overlay_test\nimport (\"testing\"; \"example.com/overlay/lib\")\nfunc TestRead(t *testing.T) { if lib.Read() != 1 { t.Fatal(\"read\") } }\n",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	extraction, err := ExtractForFix(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	overlay := map[string][]byte{filepath.Join(root, "lib/lib.go"): []byte("package lib\nfunc Read(n int) int { return n }\n")}
	if err := ValidateFixOverlay(context.Background(), extraction.Sources, overlay); err == nil {
		t.Fatal("cached importer accepted incompatible signature")
	}
	overlay[filepath.Join(root, "main.go")] = []byte(strings.ReplaceAll(files["main.go"], "lib.Read()", "lib.Read(1)"))
	if err := ValidateFixOverlay(context.Background(), extraction.Sources, overlay); err == nil {
		t.Fatal("external test caller was not checked")
	}
	overlay[filepath.Join(root, "main_test.go")] = []byte(strings.ReplaceAll(files["main_test.go"], "lib.Read()", "lib.Read(1)"))
	if err := ValidateFixOverlay(context.Background(), extraction.Sources, overlay); err != nil {
		t.Fatal(err)
	}
	next, err := ExtractFixOverlay(context.Background(), root, []string{"."}, SourceOverlay{Files: overlay, Disk: extraction.Sources})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "lib/lib.go")
	_, content, ok := next.Sources.Source(next.Sources.Reference(path))
	if !ok || content != string(overlay[path]) {
		t.Fatal("refreshed PDG uses disk bytes instead of overlay")
	}
	for name, original := range files {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(content) != original {
			t.Fatalf("preview writes %s: %v", name, err)
		}
	}
}

func TestBatchedFileQueriesPreservePackageAndTestCoverage(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"first", "second"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "code.go"), []byte("package sample\nfunc Read() int { return 1 }\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "code_test.go"), []byte("package sample\nimport \"testing\"\nfunc TestRead(t *testing.T) { _ = Read() }\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/queries\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ext, err := ExtractForFix(context.Background(), root, []string{"first/code.go", "second/code.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ext.Funcs) != 2 {
		t.Fatalf("production scope changed: %d functions", len(ext.Funcs))
	}
	for _, dir := range []string{"first", "second"} {
		if ext.Sources.Reference(filepath.Join(root, dir, "code_test.go")) == 0 {
			t.Fatal("test evidence missing")
		}
	}
}

func TestBatchQueriesDoNotActivateBuildExcludedFiles(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"active", "excluded"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "code.go"), []byte("package sample\nfunc Read() int { return 1 }\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	excluded := filepath.Join(root, "excluded", "other.go")
	if err := os.WriteFile(excluded, []byte("//go:build never_enabled\n\npackage sample\nfunc Hidden() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/excluded\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	config := &packages.Config{Context: context.Background(), Dir: root}
	query := []string{"file=" + filepath.Join(root, "active", "code.go"), "file=" + excluded}
	resolved := batchSourceQueries(config, query)
	if len(resolved) != 2 || resolved[1] != query[1] {
		t.Fatalf("excluded file query broadened to active package: %v", resolved)
	}
}

func TestValueObjectRejectsTargetsAcrossPackages(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"first", "second"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "code.go"), []byte("package sample\nfunc work(id int, name string) { println(id,name) }\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/packages\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ext, err := ExtractForFix(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	spec := &patterns.FixSpec{Kind: patterns.ValueObject, File: filepath.Join(root, "first", "code.go"), Context: ext.Sources, Params: map[string]string{"group": "id:int,name:string"}}
	for _, fn := range ext.Sources.Functions {
		if fn.Name == "work" {
			spec.Targets = append(spec.Targets, fn.ID)
		}
	}
	if _, err := PlanSignatureFix(spec); !errors.Is(err, ErrUnsupportedFix) || !strings.Contains(err.Error(), "multiple packages") {
		t.Fatalf("cross-package type rewrite accepted: %v", err)
	}
}
