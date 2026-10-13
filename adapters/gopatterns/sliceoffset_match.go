package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"go/version"

	"golang.org/x/tools/go/packages"
)

type sliceOffset struct {
	stmt  *ast.IfStmt
	lower ast.Expr
	slice *ast.SliceExpr
}

func matchSliceOffset(stmt *ast.IfStmt, info *types.Info) *sliceOffset {
	condition, ok := unparen(stmt.Cond).(*ast.BinaryExpr)
	if !ok || condition.Op != token.LAND {
		return nil
	}
	suffix := sliceOffsetSuffix(stmt.Body, info)
	if suffix == nil {
		return nil
	}
	lower := sliceOffsetBounds(condition, suffix, info)
	if lower == nil {
		return nil
	}
	if !sliceOffsetElseSupported(stmt.Else, suffix.X, info) {
		return nil
	}
	return &sliceOffset{stmt: stmt, lower: lower, slice: suffix}
}

func sliceOffsetBounds(condition *ast.BinaryExpr, suffix *ast.SliceExpr, info *types.Info) ast.Expr {
	lower := sliceOffsetLower(condition.X, suffix.Low, info)
	upper := condition.Y
	if lower == nil {
		lower = sliceOffsetLower(condition.Y, suffix.Low, info)
		upper = condition.X
	}
	if lower == nil || !sliceOffsetUpper(upper, suffix, info) {
		return nil
	}
	return lower
}

func sliceOffsetSuffix(body *ast.BlockStmt, info *types.Info) *ast.SliceExpr {
	if len(body.List) == 0 {
		return nil
	}
	assign := sliceSingleAssignment(body.List[0])
	if assign == nil {
		return nil
	}
	suffix, ok := unparen(assign.Rhs[0]).(*ast.SliceExpr)
	if !ok || !sliceIsSuffix(suffix) {
		return nil
	}
	if !sliceSameBinding(assign.Lhs[0], suffix.X, info) {
		return nil
	}
	return suffix
}

func sliceSingleAssignment(stmt ast.Stmt) *ast.AssignStmt {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || assign.Tok != token.ASSIGN {
		return nil
	}
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return nil
	}
	return assign
}

func sliceIsSuffix(slice *ast.SliceExpr) bool {
	return slice.Low != nil && slice.High == nil && !slice.Slice3
}

func sliceOffsetLower(expr, offset ast.Expr, info *types.Info) ast.Expr {
	bin, ok := unparen(expr).(*ast.BinaryExpr)
	if !ok {
		return nil
	}
	if bin.Op != token.GTR && bin.Op != token.GEQ {
		return nil
	}
	if !sliceSameBinding(bin.X, offset, info) || !sliceZero(bin.Y, info) {
		return nil
	}
	return expr
}

func sliceOffsetUpper(expr ast.Expr, suffix *ast.SliceExpr, info *types.Info) bool {
	offset, length := sliceUpperOperands(expr)
	if offset == nil {
		return false
	}
	return sliceSameBinding(offset, suffix.Low, info) && sliceLenOf(length, suffix.X, info)
}

func sliceUpperOperands(expr ast.Expr) (ast.Expr, ast.Expr) {
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

func sliceSameBinding(a, b ast.Expr, info *types.Info) bool {
	switch a := unparen(a).(type) {
	case *ast.Ident:
		other, ok := unparen(b).(*ast.Ident)
		return ok && info.ObjectOf(a) != nil && info.ObjectOf(a) == info.ObjectOf(other)
	case *ast.SelectorExpr:
		other, ok := unparen(b).(*ast.SelectorExpr)
		return ok && sliceSameSelector(a, other, info)
	default:
		return false
	}
}

func sliceSameSelector(a, b *ast.SelectorExpr, info *types.Info) bool {
	return info.ObjectOf(a.Sel) != nil && info.ObjectOf(a.Sel) == info.ObjectOf(b.Sel) && sliceSameBinding(a.X, b.X, info)
}

func sliceLenOf(expr, collection ast.Expr, info *types.Info) bool {
	call, ok := unparen(expr).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok || info.ObjectOf(name) != types.Universe.Lookup("len") {
		return false
	}
	return sliceSameBinding(call.Args[0], collection, info)
}

func sliceZero(expr ast.Expr, info *types.Info) bool {
	value := info.Types[expr].Value
	return value != nil && value.Kind() == constant.Int && constant.Sign(value) == 0
}

func sliceOffsetElseSupported(branch ast.Stmt, collection ast.Expr, info *types.Info) bool {
	if branch == nil {
		return true
	}
	return sliceOffsetEmptyElse(branch, collection, info)
}

func sliceOffsetEmptyElse(branch ast.Stmt, collection ast.Expr, info *types.Info) bool {
	body, ok := branch.(*ast.BlockStmt)
	if !ok || len(body.List) != 1 {
		return false
	}
	suffix := sliceOffsetSuffix(body, info)
	return suffix != nil && sliceSameBinding(suffix.X, collection, info) && sliceLenOf(suffix.Low, collection, info)
}

func sliceOffsetAutoSafe(guard *sliceOffset, pkg *packages.Package) bool {
	if len(guard.stmt.Body.List) != 1 || !sliceOffsetEmptyElse(guard.stmt.Else, guard.slice.X, pkg.TypesInfo) {
		return false
	}
	if pkg.Types == nil || version.Compare(pkg.Types.GoVersion(), "go1.21") < 0 {
		return false
	}
	scope := pkg.Types.Scope().Innermost(guard.stmt.Pos())
	_, obj := scope.LookupParent("min", guard.stmt.Pos())
	return obj == types.Universe.Lookup("min")
}
