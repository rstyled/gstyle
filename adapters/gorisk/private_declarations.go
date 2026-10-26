package gorisk

import (
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func separatePrivateDeclarations(fset *token.FileSet, file *ast.File, path string, functions map[declarationBody]string) (map[string]changerisk.PrivateDeclaration, error) {
	result := map[string]changerisk.PrivateDeclaration{}
	kept := make([]ast.Decl, 0, len(file.Decls))
	for _, declaration := range file.Decls {
		id, function := coveredPrivateDeclaration(fset, path, declaration, functions)
		if function == nil {
			kept = append(kept, declaration)
			continue
		}
		fingerprint, err := privateSignature(fset, function)
		if err != nil {
			return nil, err
		}
		result[id] = changerisk.PrivateDeclaration{Path: path, Name: function.Name.Name, Signature: fingerprint}
	}
	file.Decls = kept
	return result, nil
}

func coveredPrivateDeclaration(fset *token.FileSet, path string, declaration ast.Decl, functions map[declarationBody]string) (string, *ast.FuncDecl) {
	function, ok := declaration.(*ast.FuncDecl)
	if !ok || !plainPrivateFunction(function) {
		return "", nil
	}
	body, err := declarationTokens(fset, function)
	if err != nil {
		return "", nil
	}
	id, exists := functions[declarationBody{path, body}]
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

func declarationTokens(fset *token.FileSet, declaration ast.Node) (string, error) {
	var source strings.Builder
	if err := printer.Fprint(&source, fset, declaration); err != nil {
		return "", err
	}
	return tokens([]byte(source.String()))
}

type declarationBody struct{ path, body string }

func declarationIdentities(functions []changerisk.Function) map[declarationBody]string {
	result := map[declarationBody]string{}
	for _, function := range functions {
		result[declarationBody{function.Path, function.Body}] = function.ID
	}
	return result
}

func privateSignature(fset *token.FileSet, function *ast.FuncDecl) (string, error) {
	signature := *function
	signature.Body = nil
	return declarationTokens(fset, &signature)
}
