package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestEntrypointsUseGoDeclarations(t *testing.T) {
	cases := []struct{ code, role string }{
		{"package main;func main(){}", "entrypoint"},
		{"package lib;func main(){}", ""},
		{"package main;type T int;func (T) main(){}", ""},
		{"package main;func main(x int){}", ""},
		{"package main;func main()int{return 1}", ""},
		{"package main;func main[T any](){}", ""},
		{"package lib;func init(){}", "initializer"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "a.go", tc.code, 0)
			if err != nil {
				t.Fatal(err)
			}
			fn := file.Decls[len(file.Decls)-1].(*ast.FuncDecl)
			if got := declarationRole(fn, file.Name.Name); got != tc.role {
				t.Fatalf("got %q, want %q", got, tc.role)
			}
		})
	}
}
