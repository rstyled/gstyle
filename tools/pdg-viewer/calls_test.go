package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCallFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.test/calls\n\ngo 1.26.0\n"
	for path, content := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTypedCallsRespectDispatchAndShadowing(t *testing.T) {
	root := writeCallFixture(t, map[string]string{"calls.go": `package calls
import "strings"
type Reader interface { Read() int }
type A struct{}
type B struct{}
func (A) Read() int { return 1 }
func (B) Read() int { return 2 }
func Helper() int { return 3 }
func Identity[T any](v T) T { return v }
func Use(a A, r Reader, fn func() int) int {
 Helper := fn
 _ = len("x")
 _ = int(1)
 _ = strings.TrimSpace(" x ")
 return a.Read() + r.Read() + fn() + Helper() + Identity(1)
}
`})
	index, err := indexRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if index.CallResolution != "typed" {
		t.Fatalf("coverage = %s", index.CallResolution)
	}
	names := map[string]string{}
	var calls []repoCall
	for _, s := range index.Symbols {
		names[s.ID] = s.Name
		if s.Name == "Use" {
			calls = s.Calls
		}
	}
	targets := map[string]int{}
	kinds := map[string]int{}
	for _, c := range calls {
		kinds[c.Kind]++
		if c.Target != "" {
			targets[names[c.Target]]++
		}
	}
	if len(calls) != 6 || targets["A.Read"] != 1 || targets["Identity"] != 1 || len(targets) != 2 {
		t.Fatalf("calls = %+v, targets = %+v", calls, targets)
	}
	if kinds[callDynamic] != 3 || kinds[callExternal] != 1 {
		t.Fatalf("kinds = %+v", kinds)
	}
}

func TestUnavailableCallsNeverBecomeNameGuesses(t *testing.T) {
	root := writeCallFixture(t, map[string]string{
		"good/good.go":        "package good\nfunc Target(){}\nfunc Use(){Target()}\n",
		"broken/broken.go":    "package broken\nfunc Target(){}\nfunc Use(){Target(); Missing()}\n",
		"excluded.go":         "//go:build viewer_impossible_tag\n\npackage calls\nfunc Excluded(){Missing()}\n",
		"testdata/ignored.go": "package fixture\nfunc Ignored(){}\n",
	})
	index, err := indexRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if index.CallResolution != "partial" {
		t.Fatalf("coverage = %s", index.CallResolution)
	}
	for _, s := range index.Symbols {
		switch filepath.Dir(s.File) {
		case "good":
			if s.Resolution != "typed" {
				t.Fatalf("good declaration unavailable: %+v", s)
			}
		case "testdata":
			t.Fatal("fixture included")
		default:
			if s.Resolution != "unavailable" || len(s.Calls) != 0 {
				t.Fatalf("untyped calls guessed: %+v", s)
			}
		}
	}
}

func TestRepositoryScopeExcludesBuildCopiesAndNestedModules(t *testing.T) {
	root := writeCallFixture(t, map[string]string{
		"demo.go":         "package calls\nfunc Included(){}\n",
		"dist/copy.go":    "package copy\nfunc Generated(){}\n",
		"nested/go.mod":   "module example.test/other\n\ngo 1.26.0\n",
		"nested/other.go": "package other\nfunc Other(){}\n",
	})
	index, err := indexRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Symbols) != 1 || index.Symbols[0].Name != "Included" {
		t.Fatalf("wrong scope: %+v", index.Symbols)
	}
}

func TestTypedIdentityUsesDeclarationPositions(t *testing.T) {
	root := writeCallFixture(t, map[string]string{"demo.go": "package calls\nfunc A(){}; func B(){A()}\nfunc\nC(){B()}\n"})
	index, err := indexRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Symbols) != 3 {
		t.Fatalf("symbols = %d", len(index.Symbols))
	}
	a, b, c := index.Symbols[0], index.Symbols[1], index.Symbols[2]
	if a.ID == b.ID || b.Calls[0].Target != a.ID || c.Calls[0].Target != b.ID {
		t.Fatalf("declaration identities collide: %+v", index.Symbols)
	}
}
