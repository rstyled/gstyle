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

type tableIndexScan struct {
	pkg  *packages.Package
	fn   *ast.FuncDecl
	path string
	info *types.Info
}

// unchecked reports node as a composite-literal index over an unbounded
// variable, once per site.
func (s tableIndexScan) unchecked(node ast.Node) (patterns.Candidate, bool) {
	idx, ok := node.(*ast.IndexExpr)
	if !ok || !isArrayOrSliceLit(idx.X, s.info) {
		return patterns.Candidate{}, false
	}
	if s.info.Types[idx.Index].Value != nil {
		return patterns.Candidate{}, false
	}
	id, ok := unparen(idx.Index).(*ast.Ident)
	if !ok {
		return patterns.Candidate{}, false
	}
	if literalIndexBounded(s.fn, idx, s.info) {
		return patterns.Candidate{}, false
	}
	return buildLiteralIndexCandidate(s.pkg, s.fn, s.path, id.Name, idx.Pos()), true
}

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
	scan := tableIndexScan{pkg: pkg, fn: fn, path: path, info: info}
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

func isArrayOrSliceLit(e ast.Expr, info *types.Info) bool {
	lit, ok := unparen(e).(*ast.CompositeLit)
	if !ok {
		return false
	}
	switch info.TypeOf(lit).Underlying().(type) {
	case *types.Array, *types.Slice:
		return true
	default:
		return false
	}
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
