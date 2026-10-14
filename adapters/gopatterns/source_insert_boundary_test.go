package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestInsertAfterPackageBoundary(t *testing.T) {
	for _, src := range []string{
		"package p",
		"package p;func existing(){}",
		"package p // no final newline",
		"package p /* ; in comment */;func existing(){}",
		"package p // ; in comment\nfunc existing(){}",
		"package p /* multi\nline */\nfunc existing(){}",
		"package p /* multi\r\nline */\r\nfunc existing(){}",
	} {
		t.Run(src, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "p.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			out := insertAfterImports(fs, file, []byte(src), []byte("\ntype Added int\n"))
			if _, err := parser.ParseFile(token.NewFileSet(), "p.go", out, parser.ParseComments); err != nil {
				t.Fatalf("invalid output: %s\n%v", out, err)
			}
			_, _, _, pkg := typeCheck(t, string(out))
			if pkg.Scope().Lookup("Added") == nil {
				t.Fatalf("inserted type is not a declaration: %s", out)
			}
			if strings.Contains(src, "func existing") && !strings.Contains(string(out), "func existing(){}") {
				t.Fatalf("lost existing declaration: %s", out)
			}
		})
	}
}

func TestInterfaceFixSingleLinePackage(t *testing.T) {
	src := []byte(`package p;type Dog struct{};func(Dog) Sound()string{return "woof"};type Cat struct{};func(Cat) Sound()string{return "meow"}`)
	spec := &patterns.FixSpec{Kind: patterns.TraitMethod, File: "p.go", Line: 1, Definitions: []string{"p.go:1:p.Dog.Sound", "p.go:1:p.Cat.Sound"}}
	out, err := applyProjectedFix(t, spec, src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "type Trait interface") {
		t.Fatalf("interface missing: %s", out)
	}
	typeCheck(t, string(out))
}
