package cpp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func missingSourceWire(t *testing.T) []byte {
	t.Helper()
	wire := fixtureWire(t)
	for i := range wire.Value.Vertices {
		if wire.Value.Vertices[i].Label == "FILE" {
			delete(wire.Value.Vertices[i].Properties, "CONTENT")
		}
	}
	data, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCPPSourceRootRestoresVerifiedInlineFiles(t *testing.T) {
	root := t.TempDir()
	content, err := os.ReadFile("testdata/sample.cpp")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "sample.cpp")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	data := missingSourceWire(t)
	extraction, err := readFrom(context.Background(), bytes.NewReader(data), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range extraction.Functions {
		if f.Graph.Sources[0].Snapshot == 0 {
			t.Fatalf("source root does not supply %s", f.Score.Name)
		}
	}
	changed := bytes.Replace(content, []byte("return x;"), []byte("return 0;"), 1)
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readFrom(context.Background(), bytes.NewReader(data), root); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("stale source accepted: %v", err)
	}
}

func TestCPPSourceRootRejectsSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	content, err := os.ReadFile("testdata/sample.cpp")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(outside, "sample.cpp")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Symlink(path, filepath.Join(root, "sample.cpp")); err != nil {
		t.Fatal(err)
	}
	if _, err := readFrom(context.Background(), bytes.NewReader(missingSourceWire(t)), root); err == nil {
		t.Fatal("source import follows a symlink outside the explicit root")
	}
}
