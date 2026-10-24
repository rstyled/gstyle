package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"go/ast"
	"go/token"
	"go/types"
)

// EnumDispatchHit is one enum-value switch that wants to be a dispatch
// table: `switch color { case Red: doRed(); case Blue: doBlue() }` becomes
// a `map[Color]func()` lookup. The detector mirrors the fixer's criteria
// in fixgaps.go exactly, so every hit is fixable.
type EnumDispatchHit patterns.EnumDispatchHit

// findEnumDispatches scans fn for enum-value switches that the
// enum_dispatch fixer can convert to dispatch tables.
func findEnumDispatches(fn *ast.FuncDecl, fset *token.FileSet, infos ...*types.Info) []EnumDispatchHit {
	var hits []EnumDispatchHit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		numCases, ok := isEnumDispatchSwitch(sw)
		if len(infos) > 0 {
			numCases, ok = typedDispatchCases(sw, infos[0])
		}
		if ok {
			hits = append(hits, EnumDispatchHit{
				Line:     fset.PositionFor(sw.Pos(), false).Line,
				NumCases: numCases,
			})
		}
		return true
	})
	return hits
}

// isEnumDispatchSwitch reports whether sw is a value switch the fixer can
// handle: a tag (not a type assertion), 2+ value cases, each a single
// zero-arg call to a simple identifier, with an inferable key type.
func isEnumDispatchSwitch(sw *ast.SwitchStmt) (int, bool) {
	if sw.Tag == nil {
		return 0, false
	}
	if _, ok := sw.Tag.(*ast.TypeAssertExpr); ok {
		return 0, false
	}
	return validDispatchCases(sw)
}

// validDispatchCases validates the case clauses and infers the key type.
func validDispatchCases(sw *ast.SwitchStmt) (int, bool) {
	var cases int
	var keyType string
	for _, stmt := range sw.Body.List {
		cc, ok := stmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		kt, ok := validDispatchCase(cc, keyType)
		if !ok {
			return 0, false
		}
		keyType = kt
		cases++
	}
	if cases < 2 || keyType == "" {
		return 0, false
	}
	return cases, true
}

// validDispatchCase validates one case clause and updates the key type.
func validDispatchCase(cc *ast.CaseClause, keyType string) (string, bool) {
	if len(cc.List) == 0 {
		return keyType, false // default not supported
	}
	if !isDispatchCall(cc) {
		return keyType, false
	}
	return inferDispatchKeyType(cc.List[0], keyType), true
}

// isDispatchCall reports whether the case body is a single zero-arg call
// to a simple identifier, mirroring caseCallTarget in fixgaps.go.
func isDispatchCall(cc *ast.CaseClause) bool {
	if len(cc.Body) != 1 {
		return false
	}
	exprStmt, ok := cc.Body[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	_, ok = call.Fun.(*ast.Ident)
	return ok
}

// inferDispatchKeyType infers the map key type from a case value in
// Type.Value selector form, mirroring inferKeyType in fixgaps.go.
func inferDispatchKeyType(val ast.Expr, keyType string) string {
	sel, ok := val.(*ast.SelectorExpr)
	if !ok {
		return keyType
	}
	xIdent, ok := sel.X.(*ast.Ident)
	if !ok || (keyType != "" && keyType != xIdent.Name) {
		return keyType
	}
	return xIdent.Name
}

func typedDispatchCases(sw *ast.SwitchStmt, info *types.Info) (int, bool) {
	if sw.Tag == nil || !types.Comparable(info.TypeOf(sw.Tag)) {
		return 0, false
	}
	for _, stmt := range sw.Body.List {
		if !typedDispatchCase(stmt, info) {
			return 0, false
		}
	}
	return len(sw.Body.List), len(sw.Body.List) >= 2
}

func typedDispatchCase(stmt ast.Stmt, info *types.Info) bool {
	cc, ok := stmt.(*ast.CaseClause)
	if !ok || len(cc.List) != 1 || !isDispatchCall(cc) {
		return false
	}
	return info.Types[cc.List[0]].Value != nil
}
