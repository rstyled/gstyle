package patterns

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseStmt(t *testing.T, src string) ast.Stmt {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", "package p\nfunc f() {\n"+src+"\n}", 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn := f.Decls[0].(*ast.FuncDecl)
	return fn.Body.List[0]
}

func TestAntiUnifyIdentical(t *testing.T) {
	a := parseStmt(t, "x := 1 + 2")
	b := parseStmt(t, "x := 1 + 2")
	tmpl, subs := AntiUnify(a, b)
	if len(subs) != 0 {
		t.Errorf("identical statements should have no holes, got %d", len(subs))
	}
	if _, ok := tmpl.(*ast.AssignStmt); !ok {
		t.Errorf("template should be AssignStmt, got %T", tmpl)
	}
}

func TestAntiUnifyDifferentValues(t *testing.T) {
	a := parseStmt(t, "x := 1 + 2")
	b := parseStmt(t, "x := 1 + 3")
	tmpl, subs := AntiUnify(a, b)
	if len(subs) != 1 {
		t.Fatalf("expected 1 hole for differing literal, got %d", len(subs))
	}
	if subs[0].Name != "hole0" {
		t.Errorf("hole name = %q, want hole0", subs[0].Name)
	}
	// Template should be x := 1 + hole0
	assign := tmpl.(*ast.AssignStmt)
	bin := assign.Rhs[0].(*ast.BinaryExpr)
	if _, ok := bin.Y.(*ast.Ident); !ok {
		t.Errorf("RHS should be a hole ident, got %T", bin.Y)
	}
}

func TestAntiUnifyDifferentVars(t *testing.T) {
	a := parseStmt(t, "x := foo(1)")
	b := parseStmt(t, "y := foo(1)")
	tmpl, subs := AntiUnify(a, b)
	if len(subs) != 1 {
		t.Fatalf("expected 1 hole for differing variable, got %d", len(subs))
	}
	assign := tmpl.(*ast.AssignStmt)
	if _, ok := assign.Lhs[0].(*ast.Ident); !ok {
		t.Errorf("LHS should be a hole ident, got %T", assign.Lhs[0])
	}
}

func TestAntiUnifyCommonExpressionStructure(t *testing.T) {
	for _, source := range []string{"return pkg.Format(x)", "return (-values[i])", "return table[key].Value"} {
		t.Run(source, func(t *testing.T) {
			template, holes := AntiUnify(parseStmt(t, source), parseStmt(t, source))
			if len(holes) != 0 {
				t.Fatalf("common expression became opaque: %d", len(holes))
			}
			if _, ok := template.(*ast.ReturnStmt); !ok {
				t.Fatalf("return shape lost: %T", template)
			}
		})
	}
}

func TestAntiUnifyPreservesExpressionDifferences(t *testing.T) {
	pairs := [][2]string{{"return a.Value", "return a.Other"}, {"return -x", "return +x"}, {"return values[0]", "return values[1]"}}
	for _, pair := range pairs {
		_, holes := AntiUnify(parseStmt(t, pair[0]), parseStmt(t, pair[1]))
		if len(holes) != 1 {
			t.Fatalf("difference lost between %s and %s: %d", pair[0], pair[1], len(holes))
		}
	}
}
