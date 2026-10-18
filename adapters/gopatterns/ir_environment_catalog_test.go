package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/callflow"
	"os"
	"path/filepath"
	"testing"
)

func environmentCatalogModule(t *testing.T) string {
	t.Helper()
	root := crossPackageModule(t, `package stop;func Now(flag bool)func()bool{return func()bool{return flag}}`)
	files := map[string]string{"caller.go": `package example;import "example/middle";func f(){read:=middle.Run(false);if read(){panic("bad")};println("after")}`, "middle/middle.go": `package middle;import "example/stop";func Run(flag bool)func()bool{return stop.Now(flag)}`}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func TestClosureEnvironmentsAcrossExtractionModes(t *testing.T) {
	for _, mode := range []string{"normal", "bounded", "cached"} {
		t.Run(mode, func(t *testing.T) {
			root := environmentCatalogModule(t)
			var extraction *Extraction
			var err error
			switch mode {
			case "normal":
				extraction, err = Extract(context.Background(), root, []string{"."})
			case "bounded":
				extraction, err = ExtractForFix(context.Background(), root, []string{"."})
			default:
				cache := &FixExtractor{}
				extraction, err = cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
			}
			if err != nil {
				t.Fatal(err)
			}
			checkMemoryRevision(t, extraction, callflow.Returns)
			if !afterCallReachable(t, extraction, "example.f") {
				t.Fatal("environment loses normal continuation")
			}
		})
	}
}
func TestClosureEnvironmentCatalogAcrossSourceRevisions(t *testing.T) {
	root := environmentCatalogModule(t)
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, first, callflow.Returns)
	path := filepath.Join(root, "stop/stop.go")
	files := map[string][]byte{path: []byte(`package stop;func Now(flag bool)func()bool{flag=true;return func()bool{return flag}}`)}
	second, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, second, callflow.Panics)
	if afterCallReachable(t, second, "example.f") {
		t.Fatal("environment keeps impossible continuation")
	}
	files[path] = []byte(`package stop;func Now(flag bool)func()bool{return func()bool{return flag}}`)
	third, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: second.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, third, callflow.Returns)
	if !afterCallReachable(t, third, "example.f") {
		t.Fatal("environment retains stale revision")
	}
}
