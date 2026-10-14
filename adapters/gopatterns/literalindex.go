package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// findLiteralIndices flags index expressions over array or slice composite
// literals with a non-constant index and no bounds check on that index in
// the function. `[...]string{...}[s]` panics for any s outside the table.
// Detection-only: whether out-of-range values can arrive is a contract
// question, not syntax.
func findLiteralIndices(pkg *packages.Package, root string, paths []string, sources *pdg.SourceContext) []patterns.Candidate {
	var out []patterns.Candidate
	for _, file := range pkg.Syntax {
		out = append(out, literalIndicesForFile(pkg, file, root, paths)...)
	}
	return out
}

func literalIndicesForFile(pkg *packages.Package, file *ast.File, root string, paths []string) []patterns.Candidate {
	path := pkg.Fset.PositionFor(file.Pos(), false).Filename
	if !declarationPathSelected(root, path, paths) {
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	rel = filepath.ToSlash(rel)
	var out []patterns.Candidate
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		out = append(out, literalIndexCandidates(pkg, fn, rel)...)
	}
	return out
}

func literalIndexCandidates(pkg *packages.Package, fn *ast.FuncDecl, path string) []patterns.Candidate {
	info := pkg.TypesInfo
	bounded := collectBoundedIdents(fn, info)
	scan := tableIndexScan{pkg: pkg, fn: fn, path: path, info: info, bounded: bounded, seen: map[token.Pos]bool{}}
	var out []patterns.Candidate
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if site, ok := scan.unchecked(node); ok {
			out = append(out, site)
		}
		return true
	})
	return out
}

type tableIndexScan struct {
	pkg     *packages.Package
	fn      *ast.FuncDecl
	path    string
	info    *types.Info
	bounded map[types.Object]bool
	seen    map[token.Pos]bool
}

// unchecked reports node as a composite-literal index over an unbounded
// variable, once per site.
func (s tableIndexScan) unchecked(node ast.Node) (patterns.Candidate, bool) {
	idx, ok := node.(*ast.IndexExpr)
	if !ok || !isArrayOrSliceLit(idx.X) {
		return patterns.Candidate{}, false
	}
	id, ok := idx.Index.(*ast.Ident)
	if !ok {
		return patterns.Candidate{}, false
	}
	obj := s.info.ObjectOf(id)
	if s.bounded[obj] || s.seen[idx.Pos()] {
		return patterns.Candidate{}, false
	}
	s.seen[idx.Pos()] = true
	return buildLiteralIndexCandidate(s.pkg, s.fn, s.path, id.Name, idx.Pos()), true
}

// collectBoundedIdents returns objects checked against integer bounds
// (`i >= 0`, `i < n`, `i < len(s)`) anywhere in the function.
func collectBoundedIdents(fn *ast.FuncDecl, info *types.Info) map[types.Object]bool {
	bounded := map[types.Object]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		bin, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		for _, id := range boundIdentsIn(bin, info) {
			if obj := info.ObjectOf(id); isIntVar(obj) {
				bounded[obj] = true
			}
		}
		return true
	})
	return bounded
}

// boundIdentsIn returns identifiers bounded in a comparison: either side of
// `<`, `<=`, `>`, `>=` against an integer literal, len() call, or another
// expression, provided the identifier itself is not the bound.
func boundIdentsIn(bin *ast.BinaryExpr, info *types.Info) []*ast.Ident {
	switch bin.Op {
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		var out []*ast.Ident
		if id, ok := bin.X.(*ast.Ident); ok && isBoundTarget(bin.Y, info) {
			out = append(out, id)
		}
		if id, ok := bin.Y.(*ast.Ident); ok && isBoundTarget(bin.X, info) {
			out = append(out, id)
		}
		return out
	}
	return nil
}

// isBoundTarget reports whether e can serve as a bound: integer literal,
// len() call, or any non-identifier expression.
func isBoundTarget(e ast.Expr, info *types.Info) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		return true
	case *ast.CallExpr:
		return true
	case *ast.Ident:
		_ = v
		return false
	default:
		return true
	}
}

func isArrayOrSliceLit(e ast.Expr) bool {
	lit, ok := e.(*ast.CompositeLit)
	if !ok {
		return false
	}
	_, isArray := lit.Type.(*ast.ArrayType)
	return isArray
}

func buildLiteralIndexCandidate(pkg *packages.Package, fn *ast.FuncDecl, path, name string, pos token.Pos) patterns.Candidate {
	funcName := fn.Name.Name
	if fnObj, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
		funcName = patterns.FuncID(fnObj)
	}
	line := pkg.Fset.PositionFor(pos, false).Line
	return patterns.Candidate{
		Kind:        patterns.LiteralIndex,
		ScoreMilli:  800,
		Observation: "composite literal indexed by unchecked variable " + name,
		Inference:   "verify out-of-range values are rejected before the index is used; prefer a switch with default",
		CounterEvidence: []string{
			"intent unknown; no automatic fix — verify against the contract",
		},
		Sites: []patterns.Site{{
			Path: path, Line: line, EndLine: line,
			ID: funcName, Name: funcName,
		}},
	}
}
