package gorisk

import (
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/changerisk"
)

type declarationBody struct{ path, body string }

func separatePrivateDeclarations(fset *token.FileSet, file *ast.File, content []byte, functions map[declarationBody]string) (map[string]string, error) {
	result := map[string]string{}
	kept := make([]ast.Decl, 0, len(file.Decls))
	for _, declaration := range file.Decls {
		id, function := coveredPrivateDeclaration(fset, declaration, content, functions)
		if function == nil {
			kept = append(kept, declaration)
			continue
		}
		fingerprint, err := tokens(sourceText(fset, content, function.Pos(), function.Body.Pos()))
		if err != nil {
			return nil, err
		}
		result[id] = fingerprint
	}
	file.Decls = kept
	return result, nil
}

func coveredPrivateDeclaration(fset *token.FileSet, declaration ast.Decl, content []byte, functions map[declarationBody]string) (string, *ast.FuncDecl) {
	function, ok := declaration.(*ast.FuncDecl)
	if !ok || !plainPrivateFunction(function) {
		return "", nil
	}
	body, err := tokens(sourceText(fset, content, function.Pos(), function.End()))
	if err != nil {
		return "", nil
	}
	id, exists := functions[declarationBody{fset.File(function.Pos()).Name(), body}]
	if !exists {
		return "", nil
	}
	return id, function
}

func plainPrivateFunction(function *ast.FuncDecl) bool {
	if function.Body == nil || function.Recv != nil || function.Name.IsExported() {
		return false
	}
	return function.Name.Name != "init" && function.Name.Name != "_"
}

func sourceText(fset *token.FileSet, content []byte, start, end token.Pos) []byte {
	file := fset.File(start)
	return content[file.Offset(start):file.Offset(end)]
}

func declarationIdentities(functions []changerisk.Function) map[declarationBody]string {
	result := map[declarationBody]string{}
	for _, function := range functions {
		result[declarationBody{function.Path, function.Body}] = function.ID
	}
	return result
}
