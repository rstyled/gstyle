package main

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type repoIndex struct {
	Root           string        `json:"root"`
	Module         string        `json:"module"`
	Lines          int           `json:"lines"`
	CallResolution string        `json:"call_resolution"`
	Packages       []repoPackage `json:"packages"`
	Symbols        []repoSymbol  `json:"symbols"`
}
type repoPackage struct {
	ID      string   `json:"id"`
	Files   int      `json:"files"`
	Lines   int      `json:"lines"`
	Imports []string `json:"imports"`
}
type repoSymbol struct {
	Role       string     `json:"role,omitempty"`
	Resolution string     `json:"call_resolution"`
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Package    string     `json:"package"`
	File       string     `json:"file"`
	Line       int        `json:"line"`
	EndLine    int        `json:"end_line"`
	Start      int        `json:"start"`
	End        int        `json:"end"`
	Digest     string     `json:"digest"`
	Calls      []repoCall `json:"calls"`
}
type repoCall struct {
	Spelling string `json:"spelling"`
	Target   string `json:"target,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Package  string `json:"package,omitempty"`
}
type indexBuilder struct {
	index    *repoIndex
	packages map[string]*repoPackage
}

type indexedFile struct {
	path      string
	data      []byte
	tree      *ast.File
	positions *token.FileSet
}

// gstyle-allow(side_effect_density): Repository indexing walks source files and parses syntax without running code or loading every typed PDG.
func indexRepository(root string) (*repoIndex, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	module, err := moduleName(abs)
	if err != nil {
		return nil, err
	}
	index := &repoIndex{Root: abs, Module: module, CallResolution: "unavailable", Packages: []repoPackage{}, Symbols: []repoSymbol{}}
	builder := indexBuilder{index: index, packages: map[string]*repoPackage{}}
	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		return builder.visitSource(path, d, walkErr)
	})
	if err != nil {
		return nil, err
	}
	for _, p := range builder.packages {
		sort.Strings(p.Imports)
		index.Packages = append(index.Packages, *p)
	}
	sort.Slice(index.Packages, func(i, j int) bool { return index.Packages[i].ID < index.Packages[j].ID })
	if err := resolveTypedCalls(index); err != nil {
		slog.Warn("some call links unavailable", "error", err)
	}
	return index, nil
}

// gstyle-allow(side_effect_density): The index reads the repository's module declaration.
func moduleName(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return strings.Trim(fields[1], "\""), nil
		}
	}
	return "", fmt.Errorf("go.mod has no module declaration")
}

// gstyle-allow(side_effect_density): This walk callback selects source files and propagates parse and filesystem errors.
func (b indexBuilder) visitSource(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		return b.directory(path, d.Name())
	}
	if !isSource(d.Name()) {
		return nil
	}
	f, err := parseIndexFile(b.index.Root, path)
	if err != nil {
		return err
	}
	b.addIndexFile(f)
	return nil
}

// gstyle-allow(side_effect_density): The walk stays in the requested Go module and skips generated distribution trees.
func (b indexBuilder) directory(path, name string) error {
	if path == b.index.Root {
		return nil
	}
	if err := skipDirectory(name); err != nil {
		return err
	}
	_, err := os.Stat(filepath.Join(path, "go.mod"))
	if err == nil {
		return filepath.SkipDir
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}

// gstyle-allow(side_effect_density): filepath.SkipDir is an immutable walk-control sentinel, not mutable global state.
func skipDirectory(name string) error {
	if strings.HasPrefix(name, ".") {
		return filepath.SkipDir
	}
	if name == "vendor" || name == "node_modules" || name == "testdata" || name == "dist" {
		return filepath.SkipDir
	}
	return nil
}
func isSource(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// gstyle-allow(side_effect_density): Source parsing is a file-read boundary; syntax errors remain visible.
func parseIndexFile(root, path string) (indexedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return indexedFile{}, err
	}
	positions := token.NewFileSet()
	tree, err := parser.ParseFile(positions, path, data, 0)
	if err != nil {
		return indexedFile{}, err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return indexedFile{}, err
	}
	return indexedFile{rel, data, tree, positions}, nil
}

// gstyle-allow(side_effect_density): Index aggregation stores package and symbol summaries; map updates are local to the scan.
func (b indexBuilder) addIndexFile(f indexedFile) {
	pkg := b.index.Module
	dir := filepath.ToSlash(filepath.Dir(f.path))
	if dir != "." {
		pkg += "/" + dir
	}
	p := b.packages[pkg]
	if p == nil {
		p = &repoPackage{ID: pkg, Imports: []string{}}
		b.packages[pkg] = p
	}
	accumulateFile(p, f)
	b.index.Lines += strings.Count(string(f.data), "\n") + 1
	for _, decl := range f.tree.Decls {
		b.indexDeclaration(f, pkg, decl)
	}
}

// gstyle-allow(side_effect_density): This scan-owned package accumulator records file counts and imports; helper summaries include local map writes.
func accumulateFile(p *repoPackage, f indexedFile) {
	p.Files++
	p.Lines += strings.Count(string(f.data), "\n") + 1
	indexImports(p, f.tree.Imports)
}

// gstyle-allow(side_effect_density): Accumulates imports into the scan-owned package; the mutations are local index construction.
func indexImports(p *repoPackage, imports []*ast.ImportSpec) {
	for _, imp := range imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		p.Imports = appendUnique(p.Imports, path)
	}
}
func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

// gstyle-allow(side_effect_density): The syntax visitor appends function declarations to the scan-owned index.
func (b indexBuilder) indexDeclaration(f indexedFile, pkg string, decl ast.Decl) {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok {
		return
	}
	name := declarationName(fn)
	start := f.positions.Position(fn.Pos())
	end := f.positions.Position(fn.End())
	symbol := repoSymbol{ID: fmt.Sprintf("%s:%d", f.path, f.positions.Position(fn.Name.Pos()).Offset), Name: name, Role: declarationRole(fn, f.tree.Name.Name), Package: pkg, File: f.path, Line: start.Line, EndLine: end.Line, Start: start.Offset, End: end.Offset, Digest: fmt.Sprintf("%x", sha256.Sum256(f.data)), Resolution: "unavailable", Calls: []repoCall{}}
	b.index.Symbols = append(b.index.Symbols, symbol)
}
func declarationName(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return fn.Name.Name
	}
	receiver := fn.Recv.List[0].Type
	if ptr, ok := receiver.(*ast.StarExpr); ok {
		receiver = ptr.X
	}
	if id, ok := receiver.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return "(receiver)." + fn.Name.Name
}

// declarationRole recognizes language-defined startup functions, not graph roots.
func declarationRole(fn *ast.FuncDecl, packageName string) string {
	if !startupSignature(fn) {
		return ""
	}
	if fn.Name.Name == "init" {
		return "initializer"
	}
	if packageName == "main" && fn.Name.Name == "main" {
		return "entrypoint"
	}
	return ""
}
func startupSignature(fn *ast.FuncDecl) bool {
	return fn.Recv == nil && fn.Body != nil && fn.Type.TypeParams == nil && len(fn.Type.Params.List) == 0 && noResults(fn.Type.Results)
}
func noResults(results *ast.FieldList) bool { return results == nil || len(results.List) == 0 }
