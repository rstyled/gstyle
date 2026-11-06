package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryNavigationAndLazyPDG(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/demo\n\ngo 1.26.0\n")
	write("demo.go", "package demo\nfunc Feed(x int)int{return x+1}\nfunc Use(x int)int{return Feed(x)}\n")
	write("ignored_test.go", "package demo\nfunc IgnoredTestHelper(){}\n")
	index, err := indexRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Symbols) != 2 {
		t.Fatalf("symbols: %d", len(index.Symbols))
	}
	use := index.Symbols[1]
	if use.Calls[0].Target != index.Symbols[0].ID {
		t.Fatal("local typed call is missing")
	}
	if index.CallResolution != "typed" {
		t.Fatalf("resolution: %s", index.CallResolution)
	}
	app := explorer{index: index, extraction: make(chan struct{}, 1)}
	response := httptest.NewRecorder()
	app.source(response, httptest.NewRequest("GET", "/api/source?id="+use.ID, nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), "return Feed(x)") {
		t.Fatal(response.Body.String())
	}
	data, err := app.symbolPDG(context.Background(), use)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "return Feed(x)") {
		t.Fatal("PDG source was not attached")
	}
	write("demo.go", "package demo\n")
	if _, err := app.symbolSource(use); err == nil {
		t.Fatal("changed source accepted")
	}
	response = httptest.NewRecorder()
	app.source(response, httptest.NewRequest("GET", "/api/source?id=../../secret", nil))
	if response.Code != 404 {
		t.Fatal("unindexed path accepted")
	}
}
