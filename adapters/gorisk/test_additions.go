package gorisk

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func testFile(path string, content []byte) (changerisk.TestFile, bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil || specialTestComments(file) {
		return changerisk.TestFile{}, false
	}
	result := changerisk.TestFile{Package: file.Name.Name, Imports: map[string]bool{}, Declarations: map[string]string{}}
	for _, declaration := range file.Decls {
		if !addTestDeclaration(fset, declaration, content, &result) {
			return changerisk.TestFile{}, false
		}
	}
	return result, true
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

func addTestDeclaration(fset *token.FileSet, declaration ast.Decl, content []byte, result *changerisk.TestFile) bool {
	if group, ok := declaration.(*ast.GenDecl); ok && group.Tok == token.IMPORT {
		return testImports(group, result.Imports)
	}

	name, fingerprint, ok := testFunctionSource(fset, declaration, content)
	if !ok {
		return false
	}
	if _, exists := result.Declarations[name]; exists {
		return false
	}
	result.Declarations[name] = fingerprint
	return true
}

func testFunctionSource(fset *token.FileSet, declaration ast.Decl, content []byte) (string, string, bool) {
	function, ok := declaration.(*ast.FuncDecl)
	if !ok || !plainTestFunction(function) {
		return "", "", false
	}
	start := fset.PositionFor(function.Pos(), false).Offset
	end := fset.PositionFor(function.End(), false).Offset
	return function.Name.Name, string(content[start:end]), true
}

func plainTestFunction(function *ast.FuncDecl) bool {
	if function.Body == nil || function.Recv != nil {
		return false
	}
	switch function.Name.Name {
	case "init", "TestMain", "_":
		return false
	}
	return true
}

func testImports(group *ast.GenDecl, result map[string]bool) bool {
	for _, spec := range group.Specs {
		key, ok := testImportKey(spec.(*ast.ImportSpec))
		if !ok {
			return false
		}
		result[key] = true
	}
	return true
}

func testImportKey(imported *ast.ImportSpec) (string, bool) {
	path, err := strconv.Unquote(imported.Path.Value)
	if err != nil || path == "C" {
		return "", false
	}
	name := ""
	if imported.Name != nil {
		name = imported.Name.Name
	}
	if name == "." || name == "_" {
		return "", false
	}
	return name + ":" + path, true
}
