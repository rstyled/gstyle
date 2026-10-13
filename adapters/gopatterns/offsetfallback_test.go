package gopatterns

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"golang.org/x/tools/go/packages"
)

func offsetFallbackTestSource(t *testing.T, src string) []patterns.Candidate {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Scopes: map[ast.Node]*types.Scope{}}
	pkg, err := (&types.Config{GoVersion: "go1.26"}).Check("example", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	source := offsetFallbackSource{pkg: &packages.Package{Fset: fset, TypesInfo: info, Types: pkg}, path: "test.go", src: []byte(src)}
	var out []patterns.Candidate
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			out = append(out, offsetFallbackCandidates(fn, source)...)
		}
	}
	return out
}

func TestOffsetFallback(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		find, fix  bool
	}{
		{"reversed upper bound", `if offset > 0 && len(items) >= offset { items = items[offset:] }`, true, false},
		{"reversed different collection", `other := []int{}; if offset > 0 && len(other) >= offset { items = items[offset:] }`, false, false},
		{"reversed strict bound", `if offset > 0 && len(items) > offset { items = items[offset:] }`, false, false},
		{"missing fallback", `if offset > 0 && offset <= len(items) { items = items[offset:]; seen = offset }`, true, false},
		{"explicit empty suffix", `if offset > 0 && offset <= len(items) { items = items[offset:] } else { items = items[len(items):] }`, true, true},
		{"bound first", `if offset <= len(items) && offset >= 0 { items = items[offset:] }`, true, false},
		{"extra work", `if offset > 0 && offset <= len(items) { items = items[offset:]; seen++ } else { items = items[len(items):] }`, true, false},
		{"different capacity", `if offset > 0 && offset <= len(items) { items = items[offset:] } else { items = items[:0] }`, false, false},
		{"already fixed", `if offset > 0 { items = items[min(offset, len(items)):] }`, false, false},
		{"explicit rejection", `if offset > 0 && offset <= len(items) { items = items[offset:] } else { return nil }`, false, false},
		{"different collection", `other := []int{}; if offset > 0 && offset <= len(other) { items = items[offset:] }`, false, false},
		{"shadowed offset", `if offset > 0 && offset <= len(items) { offset := 1; items = items[offset:] }`, false, false},
		{"shadowed len", `len := func([]int) int { return 2 }; if offset > 0 && offset <= len(items) { items = items[offset:] }`, false, false},
		{"shadowed min", `min := func(a, b int) int { return a }; _ = min; if offset > 0 && offset <= len(items) { items = items[offset:] } else { items = items[len(items):] }`, true, false},
		{"condition comment", "if offset > 0 /* retain */ && offset <= len(items) { items = items[offset:] } else { items = items[len(items):] }", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package example\nfunc page(items []int, offset int) []int { seen := 0; _ = seen; " + tc.body + "; return items }"
			candidates := offsetFallbackTestSource(t, src)
			if (len(candidates) != 0) != tc.find {
				t.Fatalf("got %d candidates, want finding=%v", len(candidates), tc.find)
			}
			if !tc.find {
				return
			}
			c := candidates[0]
			if (c.FixSpec != nil) != tc.fix {
				t.Fatalf("fixable=%v, want %v", c.FixSpec != nil, tc.fix)
			}
			if !strings.Contains(c.PossibleRefactor, "min(offset, len(items))") {
				t.Errorf("missing proposed diff: %s", c.PossibleRefactor)
			}
		})
	}
}

func TestOffsetFallbackFixPreservesCommentsAndRejectsStaleSource(t *testing.T) {
	src := []byte(`package example
func page(items []int, offset int) []int {
 if offset > 0 && offset <= len(items) {
  // Keep the slice comment.
  items = items[offset:]
 } else { items = items[len(items):] }
 return items
}
`)
	c := offsetFallbackTestSource(t, string(src))[0]
	fixed, err := ApplyFix(c.FixSpec, src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(fixed, []byte("Keep the slice comment.")) {
		t.Fatal("fix dropped a comment")
	}
	if _, err := ApplyFix(c.FixSpec, append(src, '\n')); err == nil {
		t.Fatal("stale source accepted")
	}
	if len(offsetFallbackTestSource(t, string(fixed))) != 0 {
		t.Fatal("fixed source still reports the pattern")
	}
}
