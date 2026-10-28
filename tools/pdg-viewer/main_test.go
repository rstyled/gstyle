package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRejectsInvalidGraphs(t *testing.T) {
	for _, input := range []string{
		`null`, `[]`, `{}`, `[{}]`,
		`[{"nodes":[{"id":"x"},{"id":"x"}],"edges":[]}]`,
		`[{"nodes":[{"id":"x"}],"edges":[{"source":"x","target":"y","kind":"data"}]}]`,
		`[{"nodes":[{"id":"x"}],"edges":[{"source":"x","target":"x","kind":"bogus"}]}]`,
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := buildDocument([]byte(input), ""); err == nil {
				t.Fatal("invalid graph accepted")
			}
		})
	}
}

func TestRenderContainsSafeStandaloneScripts(t *testing.T) {
	input := `[{"nodes":[{"id":"</script><script>alert(1)</script>"}],"edges":[]}]`
	html, err := buildDocument([]byte(input), "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<script>alert(1)") {
		t.Fatal("input can inject a script")
	}
	if strings.Count(html, "</script>") != 2 {
		t.Fatal("embedded scripts close early")
	}
	if strings.Contains(html, "/* SCRIPTS */") || strings.Contains(html, "__PDG_JSON__") {
		t.Fatal("app is not assembled")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestRunPropagatesWriteError(t *testing.T) {
	err := run(strings.NewReader(`[{"nodes":[],"edges":[]}]`), failedWriter{}, "")
	if err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("got %v", err)
	}
}

func TestSourceAttachmentAndEscape(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "demo.go")
	if err := os.WriteFile(path, []byte("package demo // café\n"), 0600); err != nil {
		t.Fatal(err)
	}
	data := []byte(`[{"nodes":[],"edges":[],"sources":[{"id":"s","path":"demo.go"}]}]`)
	attached, err := attachSources(data, root)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(attached, []byte("café")) {
		t.Fatal("source content missing")
	}
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.go")
	if err := os.WriteFile(target, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "escape.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := readSource(root, "escape.go"); err == nil {
		t.Fatal("symlink escape accepted")
	}
	if _, err := readSource(root, target); err == nil {
		t.Fatal("absolute path escape accepted")
	}
}

func TestSourceIdentityRejectsStaleContent(t *testing.T) {
	s := source{Path: "demo.go", ContentIdentity: []byte(`{"status":"known","value":"wrong"}`)}
	if err := verifyIdentity(s, []byte("package demo")); err == nil {
		t.Fatal("stale source accepted")
	}
}
