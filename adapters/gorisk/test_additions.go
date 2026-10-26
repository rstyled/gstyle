package gorisk

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
)

func testFile(path string, content []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ParseComments|parser.DeclarationErrors)
	if err != nil || specialTestComments(file) {
		return false
	}
	return !slices.ContainsFunc(file.Decls, forbiddenTestDeclaration) && !slices.ContainsFunc(file.Imports, forbiddenTestImport)
}

func specialTestComments(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.Contains(comment.Text, "go:") || strings.Contains(comment.Text, "line ") {
				return true
			}
		}
	}
	return false
}

func forbiddenTestDeclaration(declaration ast.Decl) bool {
	switch item := declaration.(type) {
	case *ast.GenDecl:
		return item.Tok != token.IMPORT
	case *ast.FuncDecl:
		if item.Body == nil || item.Recv != nil {
			return true
		}
		return slices.Contains([]string{"init", "TestMain", "_"}, item.Name.Name)
	}
	return true
}

func forbiddenTestImport(imported *ast.ImportSpec) bool {
	path, err := strconv.Unquote(imported.Path.Value)
	if err != nil || path == "C" {
		return true
	}
	return imported.Name != nil && slices.Contains([]string{".", "_"}, imported.Name.Name)
}
