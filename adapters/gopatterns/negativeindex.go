package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// findNegativeIndices flags functions where an index is upper-bound checked
// against len() with no lower-bound check anywhere, while the index is used
// in an index expression. Detection-only: whether negative indices can
// actually arrive is a contract question, not syntax.
func findNegativeIndices(pkg *packages.Package, root string, paths []string, sources *pdg.SourceContext) []patterns.Candidate {
	var out []patterns.Candidate
	for _, file := range pkg.Syntax {
		out = append(out, negativeIndicesForFile(pkg, file, root, paths)...)
	}
	return out
}

func negativeIndicesForFile(pkg *packages.Package, file *ast.File, root string, paths []string) []patterns.Candidate {
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
		out = append(out, negativeIndexCandidates(pkg, fn, rel)...)
	}
	return out
}

func negativeIndexCandidates(pkg *packages.Package, fn *ast.FuncDecl, path string) []patterns.Candidate {
	info := pkg.TypesInfo
	lower := collectLowerBounded(fn, info)
	upper := collectUpperBounded(fn, info)
	var out []patterns.Candidate
	seen := map[types.Object]bool{}
	for _, id := range orderedUpperChecks(upper) {
		pos := upper[id]
		obj := info.ObjectOf(id)
		if !needsLowerBound(obj, lower, seen) {
			continue
		}
		if !indexUsed(fn, info, obj) {
			continue
		}
		seen[obj] = true
		out = append(out, buildNegativeIndexCandidate(pkg, fn, path, id.Name, pos))
	}
	return out
}

// collectLowerBounded returns objects with an explicit lower-bound check
// (`i >= 0`, `0 <= i`, `i < 0` rejection) anywhere in the function.
func collectLowerBounded(fn *ast.FuncDecl, info *types.Info) map[types.Object]bool {
	bounded := map[types.Object]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		bin, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		if id := lowerBoundIdent(bin); id != nil {
			if obj := info.ObjectOf(id); isIntVar(obj) {
				bounded[obj] = true
			}
		}
		return true
	})
	return bounded
}

// collectUpperBounded returns identifiers upper-bound checked against len()
// in a returning branch, mapped to the check position.
func collectUpperBounded(fn *ast.FuncDecl, info *types.Info) map[*ast.Ident]token.Pos {
	bounded := map[*ast.Ident]token.Pos{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		stmt, ok := node.(*ast.IfStmt)
		if !ok || !branchReturns(stmt.Body) {
			return true
		}
		recordUpperBounds(stmt, info, bounded)
		return true
	})
	return bounded
}

func recordUpperBounds(stmt *ast.IfStmt, info *types.Info, bounded map[*ast.Ident]token.Pos) {
	for _, bin := range flattenComparisons(stmt.Cond) {
		if id := upperBoundIdent(bin, info); id != nil {
			if _, dup := bounded[id]; !dup {
				bounded[id] = stmt.Pos()
			}
		}
	}
}

// Returning branches may accept in-range values or reject out-of-range
// values. Both forms carry an upper-bound check, including combined guards.
func flattenComparisons(cond ast.Expr) []*ast.BinaryExpr {
	bin, ok := unparen(cond).(*ast.BinaryExpr)
	if !ok {
		return nil
	}
	if bin.Op == token.LAND || bin.Op == token.LOR {
		return append(flattenComparisons(bin.X), flattenComparisons(bin.Y)...)
	}
	return []*ast.BinaryExpr{bin}
}

// lowerBoundIdent returns the identifier lower-bounded by bin:
// `i >= 0`, `0 <= i` (guarded) or `i < 0`, `0 > i` (rejection).
func lowerBoundIdent(bin *ast.BinaryExpr) *ast.Ident {
	switch bin.Op {
	case token.GEQ:
		return intIdentVsZero(bin.X, bin.Y)
	case token.LEQ:
		return intIdentVsZero(bin.Y, bin.X)
	case token.LSS:
		return intIdentVsZero(bin.X, bin.Y)
	case token.GTR:
		return intIdentVsZero(bin.Y, bin.X)
	}
	return nil
}

func intIdentVsZero(maybeID, maybeZero ast.Expr) *ast.Ident {
	id, ok := maybeID.(*ast.Ident)
	if !ok || !isZeroLit(maybeZero) {
		return nil
	}
	return id
}

func isZeroLit(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Value == "0"
}

// The direction distinguishes acceptance from rejection, but both are
// relevant to this advisory rule. Only the builtin len is accepted.
func upperBoundIdent(bin *ast.BinaryExpr, info *types.Info) *ast.Ident {
	switch bin.Op {
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		if id := indexComparedWithLength(bin.X, bin.Y, info); id != nil {
			return id
		}
		return indexComparedWithLength(bin.Y, bin.X, info)
	default:
		return nil
	}
}

func indexComparedWithLength(index, bound ast.Expr, info *types.Info) *ast.Ident {
	id, ok := unparen(index).(*ast.Ident)
	if !ok || !isLenCallExpr(unparen(bound), info) {
		return nil
	}
	return id
}

func isLenCallExpr(e ast.Expr, info *types.Info) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	if isLenCall(call, info) {
		return true
	}
	if !integerBoundConversion(call, info) {
		return false
	}
	return isLenCallExpr(call.Args[0], info)
}

func integerBoundConversion(call *ast.CallExpr, info *types.Info) bool {
	if len(call.Args) != 1 || !info.Types[call.Fun].IsType() {
		return false
	}
	basic, ok := info.TypeOf(call).Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsInteger != 0
}

// indexUsed reports whether obj is used in an index expression in the body.
func indexUsed(fn *ast.FuncDecl, info *types.Info, obj types.Object) bool {
	used := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		idx, ok := node.(*ast.IndexExpr)
		if !ok {
			return true
		}
		if id, ok := idx.Index.(*ast.Ident); ok && info.ObjectOf(id) == obj {
			used = true
			return false
		}
		return true
	})
	return used
}

func isIntVar(obj types.Object) bool {
	v, ok := obj.(*types.Var)
	if !ok {
		return false
	}
	basic, ok := v.Type().Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return basic.Info()&types.IsInteger != 0
}

func buildNegativeIndexCandidate(pkg *packages.Package, fn *ast.FuncDecl, path, name string, pos token.Pos) patterns.Candidate {
	funcName := fn.Name.Name
	if fnObj, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
		funcName = patterns.FuncID(fnObj)
	}
	line := pkg.Fset.PositionFor(pos, false).Line
	return patterns.Candidate{
		Kind:       patterns.NegativeIndex,
		ScoreMilli: 800,
		Observation: "upper-bound check on " + name + " has no lower-bound check; " +
			name + " indexes a collection",
		Inference: "verify negative indices are rejected or impossible before the index is used",
		CounterEvidence: []string{
			"intent unknown; no automatic fix — verify against the contract",
		},
		Sites: []patterns.Site{{
			Path: path, Line: line, EndLine: line,
			ID: funcName, Name: funcName,
		}},
	}
}

func isSignedIntVar(obj types.Object) bool {
	if !isIntVar(obj) {
		return false
	}
	basic := obj.Type().Underlying().(*types.Basic)
	return basic.Info()&types.IsUnsigned == 0
}

func needsLowerBound(obj types.Object, lower, seen map[types.Object]bool) bool {
	return !lower[obj] && !seen[obj] && isSignedIntVar(obj)
}

func orderedUpperChecks(checks map[*ast.Ident]token.Pos) []*ast.Ident {
	ids := make([]*ast.Ident, 0, len(checks))
	for id := range checks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return checks[ids[i]] < checks[ids[j]] })
	return ids
}
