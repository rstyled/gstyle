package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/types"
)

func (b *irBuilder) bindingRole(object types.Object, role pdg.Role) pdg.Role {
	if object.Pkg() != nil && object.Parent() == object.Pkg().Scope() {
		return pdg.GlobalRole
	}
	if b.capturedVariable(object) {
		return pdg.CaptureRole
	}
	return role
}

// Scope ancestry uses compiler object identity. Source spelling and byte
// positions cannot distinguish shadowed bindings or establish a missing scope.
func (b *irBuilder) capturedVariable(object types.Object) bool {
	variable, ok := object.(*types.Var)
	if !ok || variable.IsField() || packageVariable(variable) {
		return false
	}
	if b.functionScope == nil {
		return false
	}
	return !scopeWithin(variable.Parent(), b.functionScope)
}

func scopeWithin(scope, root *types.Scope) bool {
	for current := scope; current != nil; current = current.Parent() {
		if current == root {
			return true
		}
	}
	return false
}

func (b *irBuilder) captureSymbol(identifier *ast.Ident, variable *types.Var) pdg.Ref {
	if ref := b.symbols[variable]; ref != 0 {
		return ref
	}
	if !b.capturedVariable(variable) {
		return 0
	}
	return b.symbol(identifier, pdg.CaptureRole)
}

func (b *irBuilder) capturesOutsideScope(body *ast.BlockStmt) bool {
	captured := false
	ast.Inspect(body, func(node ast.Node) bool {
		if captured {
			return false
		}
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		captured = b.capturedVariable(b.info.Uses[identifier])
		return true
	})
	return captured
}

// The local profiles do not model captured storage. General binding lowering
// retains those cells and their unknown external definitions instead.
func (b *irBuilder) selectLocalProfiles(fn *ast.FuncDecl) {
	if b.capturesOutsideScope(fn.Body) {
		return
	}
	b.scalarOperations = b.selectScalarSource(fn)
	b.bindingOperations = !b.scalarOperations && b.interfaceBindingSubset(fn)
}
