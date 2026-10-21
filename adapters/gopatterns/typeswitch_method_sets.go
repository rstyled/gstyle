package gopatterns

import (
	"go/ast"
	"go/types"
)

func typeSwitchMethodSets(sw *ast.TypeSwitchStmt, info *types.Info) bool {
	if info == nil {
		return false
	}
	for _, statement := range sw.Body.List {
		clause, ok := statement.(*ast.CaseClause)
		if !ok {
			return false
		}
		if len(clause.List) == 0 {
			continue
		}
		if !typeSwitchArmMethodSet(clause, info) {
			return false
		}
	}
	return true
}

func typeSwitchArmMethodSet(clause *ast.CaseClause, info *types.Info) bool {
	call, ok := singleCallExpr(clause.Body)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := info.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal {
		return false
	}
	method := selection.Obj()
	return types.NewMethodSet(selection.Recv()).Lookup(method.Pkg(), method.Name()) != nil
}
