package gopatterns

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/adapters/pdgjson"
)

func cachedTestModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                    "module example.com/cache\n\ngo 1.24\n",
		"library/library.go":        "package library\nfunc Read() *int { return nil }\n",
		"consumer/consumer.go":      "package consumer\nimport \"example.com/cache/library\"\nfunc IsNil() bool { return library.Read() == nil }\n",
		"consumer/consumer_test.go": "package consumer_test\nimport (\"testing\";\"example.com/cache/consumer\")\nfunc TestRead(t *testing.T) { _ = consumer.IsNil() }\n",
		"unrelated/unrelated.go":    "package unrelated\nfunc Stable() int { return 2 }\n",
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
	return root
}

func graphDocuments(t *testing.T, extraction *Extraction) map[string]string {
	t.Helper()
	docs := make(map[string]string)
	for i := range extraction.Funcs {
		fn := &extraction.Funcs[i]
		var buffer bytes.Buffer
		if err := pdgjson.Write(&buffer, &fn.Pdg); err != nil {
			t.Fatal(err)
		}
		docs[fn.Name] = buffer.String()
	}
	return docs
}

func TestCachedRefreshMatchesFullExtractionAndReusesUnrelatedIR(t *testing.T) {
	root := cachedTestModule(t)
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	library := filepath.Join(root, "library", "library.go")
	overlay := map[string][]byte{library: []byte("package library\nfunc Read() any { var p *int; return p }\n")}
	cache.Invalidate([]string{library})
	next, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: overlay, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	full, err := ExtractFixOverlay(context.Background(), root, []string{"."}, SourceOverlay{Files: overlay, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	actual, expected := graphDocuments(t, next), graphDocuments(t, full)
	for name, document := range expected {
		if actual[name] != document {
			t.Fatalf("cached graph differs from full refresh: %s", name)
		}
	}
	if len(actual) != len(expected) {
		t.Fatal("cached extraction loses functions")
	}
	beforeUnrelated, afterUnrelated := findCachedFunction(first, "unrelated.Stable"), findCachedFunction(next, "unrelated.Stable")
	if &beforeUnrelated.Pdg.Nodes[0] != &afterUnrelated.Pdg.Nodes[0] {
		t.Fatal("unrelated package IR was rebuilt")
	}
	if first.Sources.Reference(library) != next.Sources.Reference(library) {
		t.Fatal("snapshot reference shifted across revisions")
	}
	if err := first.Sources.Check(library, []byte("package library\nfunc Read() *int { return nil }\n")); err != nil {
		t.Fatal("published original evidence changed")
	}
	if len(next.Sources.References) != len(full.Sources.References) || len(next.Sources.Functions) != len(full.Sources.Functions) {
		t.Fatal("cached extraction loses or duplicates test/source inventory")
	}
}

func findCachedFunction(extraction *Extraction, suffix string) *FuncPdg {
	for i := range extraction.Funcs {
		if strings.HasSuffix(extraction.Funcs[i].Name, suffix) {
			return &extraction.Funcs[i]
		}
	}
	return nil
}

func TestCacheRejectsUntrackedSourceChanges(t *testing.T) {
	root := cachedTestModule(t)
	cache := &FixExtractor{}
	if _, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unrelated", "unrelated.go"), []byte("package unrelated\nfunc Stable() int { return 3 }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{}); err == nil || !strings.Contains(err.Error(), "source changed") {
		t.Fatalf("stale unrelated IR reused: %v", err)
	}
}

func TestCacheFallsBackForNewPackageInventory(t *testing.T) {
	root := cachedTestModule(t)
	cache := &FixExtractor{}
	if _, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "unrelated", "extra.go")
	if err := os.WriteFile(path, []byte("package unrelated\nfunc Extra() int { return 4 }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	next, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	if findCachedFunction(next, "unrelated.Extra") == nil {
		t.Fatal("new source file omitted from refresh")
	}
}

func TestCacheDerivesOverlayInvalidation(t *testing.T) {
	root := cachedTestModule(t)
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "unrelated", "unrelated.go")
	overlay := map[string][]byte{path: []byte("package unrelated\nfunc Stable() int { return 8 }\n")}
	next, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: overlay, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	full, err := ExtractFixOverlay(context.Background(), root, []string{"."}, SourceOverlay{Files: overlay, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	actual, expected := graphDocuments(t, next), graphDocuments(t, full)
	name := findCachedFunction(next, "unrelated.Stable").Name
	if actual[name] != expected[name] {
		t.Fatal("virtual source change reused stale PDG")
	}
}
