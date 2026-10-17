package gopatterns

import (
	"go/ast"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestObligationHelperRelease(t *testing.T) {
	for _, tc := range []struct {
		name, helper, caller string
		want                 int
	}{
		{"return close", `return f.Close()`, `return cleanup(f)`, 0},
		{"opaque writer before joined close", `_, err := io.WriteString(f, "rules"); return errors.Join(err, f.Close())`, `if err := cleanup(f); err != nil { return err }; return nil`, 1},
		{"deferred helper close", `defer f.Close(); return nil`, `return cleanup(f)`, 0},
		{"direct close", `return nil`, `return f.Close()`, 0},
		{"both arms close", `if choose { return f.Close() }; return f.Close()`, `return cleanup(f)`, 0},
		{"caller both arms close", `return f.Close()`, `if choose { return cleanup(f) }; return cleanup(f)`, 0},
		{"conditional helper close", `if choose { return f.Close() }; return nil`, `return cleanup(f)`, 1},
		{"early helper return", `if choose { return nil }; return f.Close()`, `return cleanup(f)`, 1},
		{"early caller return", `return f.Close()`, `if choose { return nil }; return cleanup(f)`, 1},
		{"wrong helper argument", `return f.Close()`, `return cleanup(other)`, 1},
		{"helper rebind", `f = other; return f.Close()`, `return cleanup(f)`, 1},
		{"caller rebind", `return f.Close()`, `f = other; return cleanup(f)`, 1},
		{"caller error rebind", `return f.Close()`, `err = errors.New("other"); if err != nil { return nil }; return cleanup(f)`, 1},
		{"goroutine cleanup", `return f.Close()`, `go cleanup(f); return nil`, 1},
		{"closure cleanup", `return f.Close()`, `defer func() { cleanup(f) }(); return nil`, 1},
		{"loop before close", `for choose {}; return f.Close()`, `return cleanup(f)`, 1},
		{"shadowed resource", `if choose { f := other; return f.Close() }; return f.Close()`, `return cleanup(f)`, 1},
		{"alias address", `alias := &f; *alias = other; return f.Close()`, `return cleanup(f)`, 1},
		{"helper write failure skips close", `_, err := io.WriteString(f, "rules"); if err != nil { return err }; return f.Close()`, `return cleanup(f)`, 1},
		{"caller error address", `return f.Close()`, `ptr := &err; *ptr = errors.New("other"); if err != nil { return nil }; return cleanup(f)`, 1},
		{"deferred helper", `return f.Close()`, `defer cleanup(f); return nil`, 0},
		{"later abort blocks deferred close", `defer f.Close(); defer os.Exit(0); return nil`, `return cleanup(f)`, 1},
		{"close precedes abort on unwind", `defer os.Exit(0); defer f.Close(); return nil`, `return cleanup(f)`, 0},
		{"caller later abort blocks close", `return f.Close()`, `defer f.Close(); defer os.Exit(0); return nil`, 1},
		{"caller close precedes abort", `return f.Close()`, `defer os.Exit(0); defer f.Close(); return nil`, 0},
		{"print returns before older close", `defer f.Close(); defer println("exit"); return nil`, `return cleanup(f)`, 0},
		{"loop never starts exit", `defer f.Close(); for choose {}; return nil`, `return cleanup(f)`, 1},
		{"conditional abort blocks close", `defer f.Close(); if choose { defer os.Exit(0) }; return nil`, `return cleanup(f)`, 1},
		{"conditional deferred close", `if choose { defer f.Close() }; return nil`, `return cleanup(f)`, 1},
		{"branch capture stacks stay distinct", `if choose { defer f.Close() } else { defer other.Close() }; return nil`, `return cleanup(f)`, 1},
		{"direct abort does not unwind", `defer f.Close(); os.Exit(0); return nil`, `return cleanup(f)`, 1},
		{"short circuit cleanup", `return f.Close()`, `if choose && cleanup(f) != nil { return nil }; return nil`, 1},
		{"helper does not close", `return nil`, `return cleanup(f)`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package testp
import ("os"; "io"; "errors")
var choose bool
var other *os.File
var _ = io.Discard
var _ = errors.New
func caller() error {
 f, err := os.Open("rules.yml")
 if err != nil { return err }
 _ = f
 ` + tc.caller + `
}
func cleanup(f io.WriteCloser) error { ` + tc.helper + ` }
`
			pkg := loadTestPackage(t, source)
			if len(pkg.Errors) != 0 {
				t.Fatal(pkg.Errors)
			}
			root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
			functions, err := packagePdgs(pkg, root, pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			var facts []*patterns.FuncFacts
			for i := range functions {
				fp := &functions[i]
				facts = append(facts, &patterns.FuncFacts{ID: fp.Name, Name: fp.Name, Path: fp.Path, Pdg: &fp.Pdg, ReleasedFileLines: fp.ReleasedFileLines})
			}
			report := patterns.Run(facts, patterns.Options{})
			count := 0
			for _, candidate := range report.Candidates {
				if candidate.Kind == patterns.Obligation {
					count++
				}
			}
			if count != tc.want {
				t.Fatalf("got %d obligation candidates, want %d", count, tc.want)
			}
		})
	}
}

func TestHelperReleaseDoesNotDischargeSameLineAcquisitions(t *testing.T) {
	pkg := loadTestPackage(t, `package testp
import "os"
func caller() error {
 f, err := os.Open("one"); g, otherErr := os.Open("two")
 if err != nil { return err }; if otherErr != nil { return otherErr }
 _ = g
 return cleanup(f)
}
func cleanup(f *os.File) error { return f.Close() }
`)
	// Test the line-only evidence guard with the same native graph path used by mining.
	for _, decl := range pkg.Syntax[0].Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "caller" {
			continue
		}
		graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
		if err != nil {
			t.Fatal(err)
		}
		facts := &patterns.MiningFacts{ID: "caller", Pdg: patterns.MiningView(&graph), ReleasedFileLines: []int{4}}
		if got := patterns.FindObligationViolations([]*patterns.MiningFacts{facts}); len(got) != 2 {
			t.Fatalf("got %v", got)
		}
	}
}
