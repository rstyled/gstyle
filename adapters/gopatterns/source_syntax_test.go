package gopatterns

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func projectedSource(t *testing.T, path, source string) *pdg.SourceContext {
	t.Helper()
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	// A preceding file proves that positions are source-relative, not retained
	// token.FileSet IDs. The retained inventory contains no Go compiler objects.
	fset.AddFile("earlier.go", -1, 1234)
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	context := pdg.NewSourceContext(filepath.Dir(path), nil)
	if err := publishSourceSnapshot(context, []byte(source), fset, file); err != nil {
		t.Fatal(err)
	}
	return context
}

func TestSourceSyntaxRoundTrip(t *testing.T) {
	source := `// Package p has syntax coverage.
package p
import alias "fmt"
// Box holds a value.
type Box[T any] struct { Value T }
type I interface { M(...int) (int, error) }
var _ = []int{1, 2, 3}
func (b *Box[T]) M(v ...int) (result int, err error) {
 label:
 for i := range v {
  if i > 2 { break label } else if i == 2 { continue }
  v[i]++
 }
 f := func(x int) int { return -x + 1 }
 go alias.Println(f(v[0]))
 defer alias.Println("// literal /* text */")
 ch := make(chan int)
 select { case ch <- v[0]: case x := <-ch: _ = x; default: }
 switch x := any(b).(type) { case I: result, err = x.M(v...); default: }
 switch result { case 1, 2: result = len(v[:1:2]); default: }
 _ = Box[int]{Value: *new(int)}
 _ = f
 return
}
`
	context := projectedSource(t, "syntax.go", source)
	path := filepath.Join(context.Root, "syntax.go")
	fset, file, err := sourceSyntaxTree(path, context)
	if err != nil {
		t.Fatal(err)
	}
	var actual bytes.Buffer
	if err := format.Node(&actual, fset, file); err != nil {
		t.Fatal(err)
	}
	expected, err := format.Source([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual.Bytes(), expected) {
		t.Fatalf("syntax rendering differs:\n%s", actual.String())
	}
	snapshot, _ := context.Snapshot(context.Reference(path))
	t.Logf("source=%d syntax=%d bytes", len(source), len(snapshot.Syntax))
	// Mutation of a disposable tree must not affect the shared revision.
	file.Name.Name = "changed"
	_, next, err := sourceSyntaxTree(path, context)
	if err != nil || next.Name.Name != "p" {
		t.Fatalf("shared syntax was mutated: %v", err)
	}
}

func TestSourceSyntaxRejectsTruncation(t *testing.T) {
	context := projectedSource(t, "syntax.go", "package p\nfunc f() {}\n")
	snapshot, _ := context.Snapshot(context.Reference(filepath.Join(context.Root, "syntax.go")))
	for size := range len(snapshot.Syntax) {
		_, err := inflateSourceSyntax(snapshot.Syntax[:size], len(snapshot.Content))
		if err == nil {
			t.Fatalf("accepted truncation at %d", size)
		}
	}
}

func TestProjectedGuardMergeAndStaleRejection(t *testing.T) {
	source := "package p\n\nfunc a(x int) int {\n if x > 0 { println(x); println(x+1); println(x+2) } else { return -1 }; return 0\n}\nfunc b(x int) int {\n if x > 0 { println(x); println(x+1); println(x+2) } else { return -1 }; return 0\n}\n"
	formatted, err := format.Source([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	source = string(formatted)
	context := projectedSource(t, "guards.go", source)
	path := filepath.Join(context.Root, "guards.go")
	fset, file, err := sourceSyntaxTree(path, context)
	if err != nil {
		t.Fatal(err)
	}
	var specs []*patterns.FixSpec
	ast.Inspect(file, func(node ast.Node) bool {
		guard, ok := node.(*ast.IfStmt)
		if ok {
			specs = append(specs, &patterns.FixSpec{Context: context, File: path, Kind: patterns.GuardClause, Line: fset.Position(guard.Pos()).Line, Params: map[string]string{"if_line": strconv.Itoa(fset.Position(guard.Pos()).Line)}})
		}
		return true
	})
	snapshot, _ := context.Snapshot(context.Reference(path))
	result, count, skipped, err := PlanLocalFixes(specs, snapshot)
	if err != nil || count != 2 || len(skipped) != 0 {
		t.Fatalf("merged count=%d skipped=%v err=%v", count, skipped, err)
	}
	if strings.Count(string(result), "if !(x > 0)") != 2 {
		t.Fatalf("missing guard edits: %s", result)
	}
	if _, err := ApplyFix(specs[0], result); err == nil {
		t.Fatal("stale source accepted")
	}
	// Identical candidates overlap and must not write duplicate replacements.
	_, count, skipped, err = PlanLocalFixes([]*patterns.FixSpec{specs[0], specs[0]}, snapshot)
	if err != nil || count != 1 || len(skipped) != 1 {
		t.Fatalf("overlap count=%d skipped=%v err=%v", count, skipped, err)
	}
}

func TestLocalFixRequiresSyntaxEvidence(t *testing.T) {
	context := pdg.NewSourceContext(t.TempDir(), nil)
	path := filepath.Join(context.Root, "plain.go")
	context.Add(path, []byte("package p\n"), nil)
	snapshot, _ := context.Snapshot(context.Reference(path))
	spec := &patterns.FixSpec{Context: context, File: path, Kind: patterns.DeclarationOrder}
	if _, _, _, err := PlanLocalFixes([]*patterns.FixSpec{spec}, snapshot); err == nil {
		t.Fatal("missing syntax accepted")
	}
}

// Existing generator fixtures run through the shared syntax projection and
// keep their original output assertions. Invalid source remains an error.
func applyProjectedFix(t *testing.T, spec *patterns.FixSpec, source []byte) ([]byte, error) {
	t.Helper()
	if spec.Context != nil {
		return ApplyFix(spec, source)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), spec.File, source, parser.ParseComments|parser.SkipObjectResolution); err != nil {
		return nil, err
	}
	context := projectedSource(t, spec.File, string(source))
	projected := *spec
	projected.Context = context
	projected.File = filepath.Join(context.Root, filepath.Base(spec.File))
	fset, file, err := sourceSyntaxTree(projected.File, context)
	if err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	if err := format.Node(&rendered, fset, file); err != nil {
		t.Fatal(err)
	}
	expected, err := format.Source(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rendered.Bytes(), expected) {
		t.Fatalf("syntax projection differs for %s", spec.Kind)
	}
	return ApplyFix(&projected, source)
}

func TestInvalidSyntaxEvidenceStopsLocalPlan(t *testing.T) {
	context := pdg.NewSourceContext(t.TempDir(), nil)
	path := filepath.Join(context.Root, "plain.go")
	ref := context.Add(path, []byte("package p\n"), nil)
	if err := context.SetSyntax(ref, sourceSyntaxProfile, "corrupt"); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := context.Snapshot(ref)
	spec := &patterns.FixSpec{Context: context, File: path, Kind: patterns.DeclarationOrder}
	if _, _, _, err := PlanLocalFixes([]*patterns.FixSpec{spec}, snapshot); err == nil {
		t.Fatal("invalid syntax was skipped instead of stopping the plan")
	}
}

func TestLineDirectiveRewriteIsRejected(t *testing.T) {
	source := "package p\n//line other.go:123\nfunc f() {}\n"
	context := projectedSource(t, "directive.go", source)
	path := filepath.Join(context.Root, "directive.go")
	snapshot, _ := context.Snapshot(context.Reference(path))
	spec := &patterns.FixSpec{Context: context, File: path, Kind: patterns.DeclarationOrder}
	_, count, skipped, err := PlanLocalFixes([]*patterns.FixSpec{spec}, snapshot)
	if err != nil || count != 0 || len(skipped) != 1 {
		t.Fatalf("count=%d skipped=%v err=%v", count, skipped, err)
	}
}

func TestRemainingFixerProjectionParity(t *testing.T) {
	cases := []struct {
		kind   patterns.CandidateKind
		source string
		params map[string]string
	}{
		{patterns.PrimitiveObsession, "package p\nfunc SendEmail(email string) {}\nfunc ValidateEmail(email string) bool { return email != \"\" }\n", nil},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			spec := &patterns.FixSpec{Kind: tc.kind, File: "remaining.go", Params: tc.params}
			result, err := applyProjectedFix(t, spec, []byte(tc.source))
			if err != nil || string(result) == tc.source {
				t.Fatalf("no projected fix: %v", err)
			}
		})
	}
}

func TestApplyFixRequiresExtraction(t *testing.T) {
	spec := &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "plain.go"}
	if _, err := ApplyFix(spec, []byte("package p\n")); err == nil {
		t.Fatal("public fixer bypasses source inventory")
	}
}
