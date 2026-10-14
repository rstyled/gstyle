package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func fixSource(t *testing.T, src string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, _, err := FixInvertedGuards(fset, f, []byte(src))
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	return string(out)
}

func TestFixCobraExample(t *testing.T) {
	// The real inverted guard from cobra's completions.go:673.
	// Note: init uses :=, which we skip for safety (shadowing risk).
	// This test uses a non-declaring form.
	src := `package main

func f(lastArg string) string {
	if len(lastArg) > 0 {
		println("has arg")
		println(lastArg)
	} else {
		return ""
	}
	println("done")
	return lastArg
}
`
	out := fixSource(t, src)
	if !strings.Contains(out, "if !(len(lastArg) > 0) {") {
		t.Errorf("expected inverted condition, got:\n%s", out)
	}
	if !strings.Contains(out, `return ""`) {
		t.Errorf("expected early return preserved, got:\n%s", out)
	}
	// The happy path should now flow at top level (not nested).
	if strings.Contains(out, "} else {") {
		t.Errorf("else should be gone, got:\n%s", out)
	}
}

func TestFixDeMorganAnd(t *testing.T) {
	src := `package main

func f(a, b bool) int {
	if a && b {
		println("both")
		println("happy")
	} else {
		return 0
	}
	return 1
}
`
	out := fixSource(t, src)
	if !strings.Contains(out, "if !a || !b {") {
		t.Errorf("expected De Morgan inversion, got:\n%s", out)
	}
}

func TestFixDeMorganOr(t *testing.T) {
	src := `package main

func f(a, b bool) int {
	if a || b {
		println("either")
		println("happy")
	} else {
		return 0
	}
	return 1
}
`
	out := fixSource(t, src)
	if !strings.Contains(out, "if !a && !b {") {
		t.Errorf("expected De Morgan inversion, got:\n%s", out)
	}
}

func TestFixComparison(t *testing.T) {
	src := `package main

func f(x int) int {
	if x == 0 {
		println("zero")
		println("happy")
	} else {
		return -1
	}
	return x
}
`
	out := fixSource(t, src)
	if !strings.Contains(out, "if x != 0 {") {
		t.Errorf("expected != inversion, got:\n%s", out)
	}
}

func TestFixInitHoisted(t *testing.T) {
	src := `package main

import "strings"

func f(lastArg string) string {
	if index := strings.Index(lastArg, "="); index >= 0 {
		println("found")
		println(index)
	} else {
		return ""
	}
	return lastArg
}
`
	// := init is skipped for safety.
	out := fixSource(t, src)
	if strings.Contains(out, "if index < 0 {") {
		t.Errorf(":= init should be skipped, got:\n%s", out)
	}
	if !strings.Contains(out, "} else {") {
		t.Errorf("original should be unchanged, got:\n%s", out)
	}
}

func TestFixInitAssignment(t *testing.T) {
	src := `package main

func f(x int) int {
	y := 0
	if y = x + 1; y > 0 {
		println("positive")
		println(y)
	} else {
		return -1
	}
	return y
}
`
	out := fixSource(t, src)
	if !strings.Contains(out, "y = x + 1\n") {
		t.Errorf("expected init hoisted, got:\n%s", out)
	}
	if !strings.Contains(out, "if !(y > 0) {") {
		t.Errorf("expected inverted condition, got:\n%s", out)
	}
}

func TestFixNoInvertedGuard(t *testing.T) {
	src := `package main

func f(x int) int {
	if x < 0 {
		return -1
	}
	println("happy")
	return x
}
`
	out := fixSource(t, src)
	if out != src && !strings.Contains(out, "if x < 0 {") {
		t.Errorf("proper guard should be untouched, got:\n%s", out)
	}
}

func TestFixIdempotent(t *testing.T) {
	src := `package main

func f(x int) int {
	if x > 0 {
		println("positive")
		println(x)
	} else {
		return -1
	}
	return x
}
`
	once := fixSource(t, src)
	twice := fixSource(t, once)
	if once != twice {
		t.Errorf("not idempotent:\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
}

func TestGuardFixCommentMarkersInLiterals(t *testing.T) {
	for _, literal := range []string{`"/*"`, `"//"`, "`/* raw */`"} {
		source := "package p\nfunc classify(value string) int {\n if value == " + literal + " {\n value += \"accepted\"\n println(value)\n } else { return 0 }\n return 1\n}\n"
		output := fixSource(t, source)
		if strings.Contains(output, "} else {") {
			t.Fatalf("literal mistaken for comment: %s", output)
		}
		if !strings.Contains(output, literal) {
			t.Fatal("literal lost")
		}
	}
}

func TestGuardFixPreservesActualConditionComments(t *testing.T) {
	source := "package p\nfunc classify(value string) int {\n if value /* preserve */ == \"/*\" {\n value += \"accepted\"\n println(value)\n } else { return 0 }\n return 1\n}\n"
	if output := fixSource(t, source); output != source {
		t.Fatalf("condition comment changed: %s", output)
	}
}

func TestGuardFixNestedElseIfKeepsHappyPathConditional(t *testing.T) {
	source := `package p
func classify(first, second bool) int {
 result := 0
 if first {
  result = 1
 } else if second {
  result = 2
  result += 10
 } else {
  return 3
 }
 return result
}
`
	output := fixSource(t, source)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixed.go", output, 0)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	outer := fn.Body.List[1].(*ast.IfStmt)
	branch, ok := outer.Else.(*ast.BlockStmt)
	if !ok {
		t.Fatalf("else-if remains unscoped: %s", output)
	}
	if len(branch.List) != 3 {
		t.Fatalf("happy path escapes else: %s", output)
	}
	guard := branch.List[0].(*ast.IfStmt)
	if guard.Else != nil || !blockEndsReturn(guard.Body) {
		t.Fatal("missing terminal guard")
	}
	if len(fn.Body.List) != 3 {
		t.Fatal("happy path moved outside conditional branch")
	}
	info := &types.Info{}
	config := types.Config{}
	if _, err := config.Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
}

func TestApplyGuardLiteralElseIfSpec(t *testing.T) {
	source := `package p
func classify(value string) int {
 if value == "line" { return 1 } else if value == "/*" {
  value += "block"
  println(value)
 } else { return 0 }
 return 2
}
`
	spec := &patterns.FixSpec{Kind: patterns.GuardClause, File: "sample.go", Params: map[string]string{"if_line": "3"}}
	// Two ifs share a line: the spec must select the matching inner guard.
	output, err := applyProjectedFix(t, spec, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "else {") || !strings.Contains(string(output), `if value != "/*"`) {
		t.Fatalf("incorrect nested guard: %s", output)
	}
}

func TestGuardFixPreservesHeaderComments(t *testing.T) {
	for _, condition := range []string{`/* before */ value == "/*"`, `value == "/*" /* after */`} {
		source := "package p\nfunc classify(value string) int {\n if " + condition + " {\n value += \"accepted\"\n println(value)\n } else { return 0 }\n return 1\n}\n"
		if output := fixSource(t, source); output != source {
			t.Fatalf("header comment lost: %s", output)
		}
	}
}
