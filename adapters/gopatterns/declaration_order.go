package gopatterns

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"golang.org/x/tools/go/packages"
)

// declarationBlock keeps source text, including comments, with its declaration.
type declarationBlock struct {
	decl  ast.Decl
	text  []byte
	owner string
	rank  int
}

func findDeclarationOrders(pkg *packages.Package, root string, paths []string) []patterns.Candidate {
	var out []patterns.Candidate
	for _, file := range pkg.Syntax {
		path := pkg.Fset.Position(file.Pos()).Filename
		if !declarationPathSelected(root, path, paths) {
			continue
		}
		candidate, ok := declarationOrderForFile(pkg.Fset, file, root, path)
		if ok {
			out = append(out, candidate)
		}
	}
	return out
}

func declarationOrderForFile(fset *token.FileSet, file *ast.File, root, path string) (patterns.Candidate, bool) {
	src, err := os.ReadFile(path)
	if err != nil {
		return patterns.Candidate{}, false
	}
	spec := &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: path}
	fixed, err := applyDeclarationOrderFix(spec, src)
	if err != nil || bytes.Equal(src, fixed) {
		return patterns.Candidate{}, false
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return patterns.Candidate{}, false
	}
	spec.File = relative
	spec.Line = fset.Position(file.Pos()).Line
	spec.EndLine = fset.Position(file.End()).Line
	return declarationOrderCandidate(spec), true
}

func declarationOrderCandidate(spec *patterns.FixSpec) patterns.Candidate {
	return patterns.Candidate{
		Kind: patterns.DeclarationOrder, ScoreMilli: 500,
		Observation:      "constructors, exported methods, private methods, and free functions are out of order",
		Inference:        "type groups are harder to scan",
		PossibleRefactor: "group each type with its constructors, exported methods, then private methods; put other functions last",
		Sites:            []patterns.Site{{Path: spec.File, Line: spec.Line, EndLine: spec.EndLine, ID: spec.File, Name: "declarations"}},
		FixSpec:          spec,
	}
}

func applyDeclarationOrderFix(spec *patterns.FixSpec, src []byte) ([]byte, error) {
	fset, file, err := parseSpec(spec, src)
	if err != nil {
		return nil, err
	}
	// Directives and cgo preambles can have adjacency requirements. Leave them intact.
	if !safeDeclarationOrder(fset, file, src) {
		return src, nil
	}
	blocks, prefix, suffix := declarationBlocks(fset, file, src)
	normalizeConstructorOwners(blocks, file)
	ordered := orderDeclarationBlocks(blocks)
	if sameDeclarationOrder(blocks, ordered) {
		return src, nil
	}
	out := append([]byte{}, prefix...)
	for _, block := range ordered {
		out = append(out, block.text...)
	}
	out = append(out, suffix...)
	return format.Source(out)
}

func hasCgoImport(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path.Value == `"C"` {
			return true
		}
	}
	return false
}

func declarationBlocks(fset *token.FileSet, file *ast.File, src []byte) ([]declarationBlock, []byte, []byte) {
	if len(file.Decls) == 0 {
		return nil, src, nil
	}
	start := declarationStart(fset, file.Decls[0])
	var blocks []declarationBlock
	cursor := start
	for _, decl := range file.Decls {
		end := declarationLineEnd(src, fset.Position(decl.End()).Offset)
		owner, rank := declarationOwner(decl)
		blocks = append(blocks, declarationBlock{decl: decl, text: src[cursor:end], owner: owner, rank: rank})
		cursor = end
	}
	return blocks, src[:start], src[cursor:]
}

func declarationStart(fset *token.FileSet, decl ast.Decl) int {
	pos := decl.Pos()
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Doc != nil {
			pos = d.Doc.Pos()
		}
	case *ast.GenDecl:
		if d.Doc != nil {
			pos = d.Doc.Pos()
		}
	}
	return fset.Position(pos).Offset
}

func declarationLineEnd(src []byte, offset int) int {
	newline := bytes.IndexByte(src[offset:], '\n')
	if newline < 0 {
		return len(src)
	}
	return offset + newline + 1
}

func declarationOwner(decl ast.Decl) (string, int) {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok {
		return "", -1
	}
	if fn.Recv != nil {
		rank := 2
		if ast.IsExported(fn.Name.Name) {
			rank = 1
		}
		return localTypeName(fn.Recv.List[0].Type), rank
	}
	if isOrderConstructorName(fn.Name.Name) && fn.Type.Results != nil {
		return constructorOwner(fn), 0
	}
	return "", 3
}

func isOrderConstructorName(name string) bool {
	return strings.HasPrefix(name, "New") || strings.HasPrefix(name, "new")
}

func constructorOwner(fn *ast.FuncDecl) string {
	if len(fn.Type.Results.List) == 0 {
		return ""
	}
	return localTypeName(fn.Type.Results.List[0].Type)
}

func localTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return localTypeName(t.X)
	case *ast.IndexExpr:
		return localTypeName(t.X)
	case *ast.IndexListExpr:
		return localTypeName(t.X)
	}
	return ""
}

func orderDeclarationBlocks(blocks []declarationBlock) []declarationBlock {
	groups := declarationGroups(blocks)
	var out []declarationBlock
	emitted := make(map[string]bool)
	for _, block := range blocks {
		if _, ok := block.decl.(*ast.FuncDecl); ok {
			continue
		}
		out = append(out, block)
		out = append(out, groupsAfterType(block.decl, groups, emitted)...)
	}
	out = append(out, remainingDeclarationGroups(blocks, groups, emitted)...)
	out = append(out, freeDeclarationFunctions(blocks)...)
	return out
}

// Receiver types can live in another file. Preserve first-seen group order.
func remainingDeclarationGroups(blocks []declarationBlock, groups map[string][]declarationBlock, emitted map[string]bool) []declarationBlock {
	var out []declarationBlock
	for _, block := range blocks {
		if block.owner == "" || emitted[block.owner] {
			continue
		}
		out = append(out, groups[block.owner]...)
		emitted[block.owner] = true
	}
	return out
}

func freeDeclarationFunctions(blocks []declarationBlock) []declarationBlock {
	var out []declarationBlock
	for _, block := range blocks {
		if _, ok := block.decl.(*ast.FuncDecl); ok && block.owner == "" {
			out = append(out, block)
		}
	}
	return out
}

func declarationGroups(blocks []declarationBlock) map[string][]declarationBlock {
	groups := make(map[string][]declarationBlock)
	for _, block := range blocks {
		if block.owner != "" {
			groups[block.owner] = append(groups[block.owner], block)
		}
	}
	for owner, group := range groups {
		sort.SliceStable(group, func(i, j int) bool { return group[i].rank < group[j].rank })
		groups[owner] = group
	}
	return groups
}

func groupsAfterType(decl ast.Decl, groups map[string][]declarationBlock, emitted map[string]bool) []declarationBlock {
	gen, ok := decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.TYPE {
		return nil
	}
	var out []declarationBlock
	for _, spec := range gen.Specs {
		name := spec.(*ast.TypeSpec).Name.Name
		if emitted[name] {
			continue
		}
		out = append(out, groups[name]...)
		emitted[name] = true
	}
	return out
}

func sameDeclarationOrder(a, b []declarationBlock) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].decl != b[i].decl {
			return false
		}
	}
	return true
}

// Same-line declarations cannot be moved as independent source blocks.
func separateDeclarationLines(fset *token.FileSet, file *ast.File) bool {
	endLine := 0
	for _, decl := range file.Decls {
		startLine := fset.Position(decl.Pos()).Line
		if startLine <= endLine {
			return false
		}
		endLine = fset.Position(decl.End()).Line
	}
	return true
}

// A New-prefixed function must return a local named type, not a scalar or
// an imported type. Methods identify types declared in another source file.
func normalizeConstructorOwners(blocks []declarationBlock, file *ast.File) {
	known := make(map[string]bool)
	for _, decl := range file.Decls {
		collectDeclarationTypes(decl, known)
	}
	for i := range blocks {
		if blocks[i].rank == 0 && !known[blocks[i].owner] {
			blocks[i].owner = ""
			blocks[i].rank = 3
		}
	}
}

func collectDeclarationTypes(decl ast.Decl, known map[string]bool) {
	switch d := decl.(type) {
	case *ast.GenDecl:
		if d.Tok != token.TYPE {
			return
		}
		for _, spec := range d.Specs {
			known[spec.(*ast.TypeSpec).Name.Name] = true
		}
	case *ast.FuncDecl:
		if d.Recv != nil {
			known[localTypeName(d.Recv.List[0].Type)] = true
		}
	}
}

func safeDeclarationOrder(fset *token.FileSet, file *ast.File, src []byte) bool {
	if bytes.Contains(src, []byte("//go:")) || hasCgoImport(file) {
		return false
	}
	return separateDeclarationLines(fset, file)
}

// Package loading includes neighboring files; explicit file input must not
// turn a declaration-order fix into a rewrite of the whole package.
func declarationPathSelected(root, path string, paths []string) bool {
	if len(paths) == 0 {
		return true
	}
	for _, input := range paths {
		if declarationInputContains(root, path, input) {
			return true
		}
	}
	return false
}

func declarationInputContains(root, path, input string) bool {
	absolute := input
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, input)
	}
	if filepath.Clean(path) == filepath.Clean(absolute) {
		return true
	}
	if strings.HasSuffix(input, ".go") {
		return false
	}
	return strings.HasPrefix(path, filepath.Clean(absolute)+string(filepath.Separator))
}
