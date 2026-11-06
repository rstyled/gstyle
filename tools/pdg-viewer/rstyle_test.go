package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRstyleCapturedGraphKeepsRealEvidence(t *testing.T) {
	root, err := filepath.Abs("testdata/rstyle-module")
	if err != nil {
		t.Fatal(err)
	}
	index, err := indexRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := loadRstyle("testdata/rstyle-capture", "testdata/rstyle-capture/report.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range index.Symbols {
		data, err := store.document(s)
		if err != nil {
			t.Fatal(err)
		}
		var g []rstyleGraph
		if err := json.Unmarshal(data, &g); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(g[0].Flow), `"standing":"known"`) {
			t.Fatal("flow standing was lost")
		}
		if s.Name == "Forward" && (len(g[0].Diagnostics) != 1 || !strings.Contains(string(g[0].Diagnostics[0]), `"lines":[6,7]`)) {
			t.Fatalf("mutation evidence lost: %s", data)
		}
		stale := s
		stale.Digest = "changed"
		if _, err := store.document(stale); err == nil {
			t.Fatal("stale CPG snapshot accepted")
		}
	}
}
func TestRstyleRejectsIncompleteAndCorruptCaptures(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "documents"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile("testdata/rstyle-capture/documents/documents.manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "documents", "documents.manifest.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "documents", "document-000000.cpg.json")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRstyle(root, "testdata/rstyle-capture/report.json"); err == nil || !strings.Contains(err.Error(), "seal mismatch") {
		t.Fatalf("corrupt capture: %v", err)
	}
	incomplete := filepath.Join(root, "report.json")
	if err := os.WriteFile(incomplete, []byte(`{"check":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRstyle(root, incomplete); err == nil {
		t.Fatal("missing analyses accepted")
	}
}
func TestRstyleFlowIDsAreScopedBySource(t *testing.T) {
	report := rstyleReport{}
	if err := decodeRstyle("testdata/rstyle-capture/report.json", &report); err != nil {
		t.Fatal(err)
	}
	report.Flow.Functions = append([]rstyleFlow{{Function: "f0", Path: "another.go", Flow: json.RawMessage(`{"standing":"unknown"}`)}}, report.Flow.Functions...)
	flow, err := report.flow("flow.go", "f0")
	if err != nil || !strings.Contains(string(flow), `"known"`) {
		t.Fatalf("unit-local flow IDs collided: %s %v", flow, err)
	}
}
func TestRstyleLiveCapture(t *testing.T) {
	binary := os.Getenv("RSTYLE_VIEWER_TEST_BIN")
	if binary == "" {
		t.Skip("set RSTYLE_VIEWER_TEST_BIN to exercise the installed backend")
	}
	root, err := filepath.Abs("testdata/rstyle-module")
	if err != nil {
		t.Fatal(err)
	}
	store, err := captureRstyle(root, binary)
	if err != nil {
		t.Fatal(err)
	}
	if len(store) != 2 {
		t.Fatalf("got %d functions, want 2", len(store))
	}
}
