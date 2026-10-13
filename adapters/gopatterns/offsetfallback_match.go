package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"go/version"

	"golang.org/x/tools/go/packages"
)

type offsetFallback struct {
	stmt  *ast.IfStmt
	lower ast.Expr
	slice *ast.SliceExpr
}

func matchOffsetFallback(stmt *ast.IfStmt, info *types.Info) *offsetFallback {
	condition, ok := unparen(stmt.Cond).(*ast.BinaryExpr)
	if !ok || condition.Op != token.LAND {
		return nil
	}
	suffix := offsetFallbackSuffix(stmt.Body, info)
	if suffix == nil {
		return nil
	}
	lower := offsetFallbackBounds(condition, suffix, info)
	if lower == nil {
		return nil
	}
	if !offsetFallbackElseSupported(stmt.Else, suffix.X, info) {
		return nil
	}
	return &offsetFallback{stmt: stmt, lower: lower, slice: suffix}
}

func offsetFallbackBounds(condition *ast.BinaryExpr, suffix *ast.SliceExpr, info *types.Info) ast.Expr {
	lower := offsetFallbackLower(condition.X, suffix.Low, info)
	upper := condition.Y
	if lower == nil {
		lower = offsetFallbackLower(condition.Y, suffix.Low, info)
		upper = condition.X
	}
	if lower == nil || !offsetFallbackUpper(upper, suffix, info) {
		return nil
	}
	return lower
}

func offsetFallbackSuffix(body *ast.BlockStmt, info *types.Info) *ast.SliceExpr {
	if len(body.List) == 0 {
		return nil
	}
	assign := offsetSingleAssignment(body.List[0])
	if assign == nil {
		return nil
	}
	suffix, ok := unparen(assign.Rhs[0]).(*ast.SliceExpr)
	if !ok || !offsetIsSuffix(suffix) {
		return nil
	}
	if !offsetSameBinding(assign.Lhs[0], suffix.X, info) {
		return nil
	}
	return suffix
}

func offsetSingleAssignment(stmt ast.Stmt) *ast.AssignStmt {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || assign.Tok != token.ASSIGN {
		return nil
	}
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return nil
	}
	return assign
}

func offsetIsSuffix(slice *ast.SliceExpr) bool {
	return slice.Low != nil && slice.High == nil && !slice.Slice3
}

func offsetFallbackLower(expr, offset ast.Expr, info *types.Info) ast.Expr {
	bin, ok := unparen(expr).(*ast.BinaryExpr)
	if !ok {
		return nil
	}
	if bin.Op != token.GTR && bin.Op != token.GEQ {
		return nil
	}
	if !offsetSameBinding(bin.X, offset, info) || !offsetZero(bin.Y, info) {
		return nil
	}
	return expr
}

func offsetFallbackUpper(expr ast.Expr, suffix *ast.SliceExpr, info *types.Info) bool {
	offset, length := offsetUpperOperands(expr)
	if offset == nil {
		return false
	}
	return offsetSameBinding(offset, suffix.Low, info) && offsetLenOf(length, suffix.X, info)
}

func offsetUpperOperands(expr ast.Expr) (ast.Expr, ast.Expr) {
	bin, ok := unparen(expr).(*ast.BinaryExpr)
	if !ok {
		return nil, nil
	}
	switch bin.Op {
	case token.LEQ:
		return bin.X, bin.Y
	case token.GEQ:
		return bin.Y, bin.X
	default:
		return nil, nil
	}
}

func offsetSameBinding(a, b ast.Expr, info *types.Info) bool {
	switch a := unparen(a).(type) {
	case *ast.Ident:
		other, ok := unparen(b).(*ast.Ident)
		return ok && info.ObjectOf(a) != nil && info.ObjectOf(a) == info.ObjectOf(other)
	case *ast.SelectorExpr:
		other, ok := unparen(b).(*ast.SelectorExpr)
		return ok && offsetSameSelector(a, other, info)
	default:
		return false
	}
}

func offsetSameSelector(a, b *ast.SelectorExpr, info *types.Info) bool {
	return info.ObjectOf(a.Sel) != nil && info.ObjectOf(a.Sel) == info.ObjectOf(b.Sel) && offsetSameBinding(a.X, b.X, info)
}

func offsetLenOf(expr, collection ast.Expr, info *types.Info) bool {
	call, ok := unparen(expr).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok || info.ObjectOf(name) != types.Universe.Lookup("len") {
		return false
	}
	return offsetSameBinding(call.Args[0], collection, info)
}

func offsetZero(expr ast.Expr, info *types.Info) bool {
	value := info.Types[expr].Value
	return value != nil && value.Kind() == constant.Int && constant.Sign(value) == 0
}

func offsetFallbackElseSupported(branch ast.Stmt, collection ast.Expr, info *types.Info) bool {
	if branch == nil {
		return true
	}
	return offsetFallbackEmptyElse(branch, collection, info)
}

func offsetFallbackEmptyElse(branch ast.Stmt, collection ast.Expr, info *types.Info) bool {
	body, ok := branch.(*ast.BlockStmt)
	if !ok || len(body.List) != 1 {
		return false
	}
	suffix := offsetFallbackSuffix(body, info)
	return suffix != nil && offsetSameBinding(suffix.X, collection, info) && offsetLenOf(suffix.Low, collection, info)
}

func offsetFallbackAutoSafe(guard *offsetFallback, pkg *packages.Package) bool {
	if len(guard.stmt.Body.List) != 1 || !offsetFallbackEmptyElse(guard.stmt.Else, guard.slice.X, pkg.TypesInfo) {
		return false
	}
	if pkg.Types == nil || version.Compare(pkg.Types.GoVersion(), "go1.21") < 0 {
		return false
	}
	scope := pkg.Types.Scope().Innermost(guard.stmt.Pos())
	_, obj := scope.LookupParent("min", guard.stmt.Pos())
	return obj == types.Universe.Lookup("min")
}
