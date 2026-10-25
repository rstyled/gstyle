package gocyclo

import "go/ast"

func normalizeMethodIdentities(file *ast.File) {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Obj != nil {
			continue
		}
		function.Name.Obj = &ast.Object{Kind: ast.Fun, Name: function.Name.Name, Decl: function}
	}
}
