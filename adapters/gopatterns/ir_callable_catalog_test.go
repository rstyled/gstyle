package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/callflow"
	"os"
	"path/filepath"
	"testing"
)

const callableCatalogCaller = `package example;import "example/middle";func no()bool{return false};func f(){if middle.Run(no){panic("bad")};println("after")}`

func callableCatalogModule(t *testing.T) string {
	t.Helper()
	root := crossPackageModule(t, `package stop;func Now(cb func()bool)bool{return cb()}`)
	files := map[string]string{"caller.go": callableCatalogCaller, "middle/middle.go": `package middle;import "example/stop";func Run(cb func()bool)bool{return stop.Now(cb)}`}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func TestCallableCatalogAcrossExtractionModes(t *testing.T) {
	for _, mode := range []string{"normal", "bounded", "cached"} {
		t.Run(mode, func(t *testing.T) {
			root := callableCatalogModule(t)
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
				t.Fatal("callback return loses normal continuation")
			}
		})
	}
}
func TestCallableCatalogAcrossSourceRevisions(t *testing.T) {
	root := callableCatalogModule(t)
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, first, callflow.Returns)
	files := map[string][]byte{filepath.Join(root, "caller.go"): []byte(`package example;import "example/middle";func no()bool{return true};func f(){if middle.Run(no){panic("bad")};println("after")}`)}
	second, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, second, callflow.Panics)
	if afterCallReachable(t, second, "example.f") {
		t.Fatal("callback revision retains impossible continuation")
	}
	files[filepath.Join(root, "caller.go")] = []byte(callableCatalogCaller)
	third, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: second.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, third, callflow.Returns)
	if !afterCallReachable(t, third, "example.f") {
		t.Fatal("callback revision retains stale panic")
	}
}
