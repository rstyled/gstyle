package gopatterns

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// declarationBlock keeps source text, including comments, with its declaration.
type declarationBlock struct {
	decl   ast.Decl
	text   []byte
	owner  string
	rank   int
	anchor int
}

func findDeclarationOrders(pkg *packages.Package, root string, paths []string, sources *pdg.SourceContext) []patterns.Candidate {
	var out []patterns.Candidate
	for _, file := range pkg.Syntax {
		path := pkg.Fset.Position(file.Pos()).Filename
		if !declarationPathSelected(root, path, paths) {
			continue
		}
		candidate, ok := declarationOrderForFile(pkg.Fset, file, root, sources)
		if ok {
			out = append(out, candidate)
		}
	}
	return out
}

func declarationOrderForFile(fset *token.FileSet, file *ast.File, root string, sources *pdg.SourceContext) (patterns.Candidate, bool) {
	path := fset.PositionFor(file.Pos(), false).Filename
	src, err := revisionSource(path, sources)
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
	ordered := orderDeclarationBlocks(blocks)
	if sameDeclarationOrder(blocks, ordered) {
		return src, nil
	}
	parts := make([][]byte, len(ordered)+2)
	parts[0] = prefix
	for i, block := range ordered {
		parts[i+1] = block.text
	}
	parts[len(parts)-1] = suffix
	return format.Source(bytes.Join(parts, []byte("\n")))
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
	blocks := make([]declarationBlock, len(file.Decls))
	cursor := start
	for i, decl := range file.Decls {
		end := declarationLineEnd(src, fset.Position(decl.End()).Offset)
		owner, rank := declarationOwner(decl)
		blocks[i] = declarationBlock{decl: decl, text: src[cursor:end], owner: owner, rank: rank}
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
	for expr != nil {
		if name, ok := expr.(*ast.Ident); ok {
			return name.Name
		}
		expr = localTypeBase(expr)
	}
	return ""
}

func localTypeBase(expr ast.Expr) ast.Expr {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return t.X
	case *ast.IndexExpr:
		return t.X
	case *ast.IndexListExpr:
		return t.X
	}
	return nil
}

// Plan one stable order over a copy. There is no emitted-group state and
// each declaration appears exactly once, including grouped type declarations.
func orderDeclarationBlocks(blocks []declarationBlock) []declarationBlock {
	anchors := declarationAnchors(blocks)
	ordered := append([]declarationBlock(nil), blocks...)
	for i, block := range blocks {
		ordered[i] = plannedDeclaration(block, i, len(blocks), anchors)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].anchor != ordered[j].anchor {
			return ordered[i].anchor < ordered[j].anchor
		}
		return ordered[i].rank < ordered[j].rank
	})
	return ordered
}

func plannedDeclaration(block declarationBlock, index, size int, anchors map[string]int) declarationBlock {
	if block.rank < 0 {
		return declarationBlock{decl: block.decl, text: block.text, rank: -1, anchor: index * (size + 1)}
	}
	anchor, ok := anchors[block.owner]
	rank := block.rank
	if !ok {
		anchor = 2 * size * (size + 1)
		rank = 3
	}
	return declarationBlock{decl: block.decl, text: block.text, owner: block.owner, rank: rank, anchor: anchor}
}

func declarationAnchors(blocks []declarationBlock) map[string]int {
	anchors := make(map[string]int)
	for i, block := range blocks {
		for j, name := range declaredTypeNames(block.decl) {
			anchors[name] = i*(len(blocks)+1) + j
		}
	}
	for i, block := range blocks {
		switch block.rank {
		case 1, 2:
			if _, exists := anchors[block.owner]; !exists {
				anchors[block.owner] = (len(blocks) + i) * (len(blocks) + 1)
			}
		}
	}
	return anchors
}

func declaredTypeNames(decl ast.Decl) []string {
	gen, ok := decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.TYPE {
		return nil
	}
	names := make([]string, len(gen.Specs))
	for i, spec := range gen.Specs {
		names[i] = spec.(*ast.TypeSpec).Name.Name
	}
	return names
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
