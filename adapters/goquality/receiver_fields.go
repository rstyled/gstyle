package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Closed roots do not prove ownership of their fields. Only fresh container
// headers whose values never escape receive this separate storage proof.
func closedReceiverFields(pkg *packages.Package) map[types.Object]bool {
	proof := receiverFields{receiverProof: receiverProof{pkg: pkg, parents: map[ast.Node]ast.Node{}}, owned: map[types.Object]bool{}}
	closed := closedReceiverTypes(pkg)
	for object := range closed {
		shape := object.Type().Underlying().(*types.Struct)
		for field := range shape.Fields() {
			switch field.Type().Underlying().(type) {
			case *types.Map, *types.Slice:
				proof.owned[field] = true
			}
		}
	}
	for _, file := range pkg.Syntax {
		indexParents(file, proof.parents)
	}
	for _, file := range pkg.Syntax {
		ast.Inspect(file, proof.check)
	}
	return proof.owned
}

type receiverFields struct {
	receiverProof
	owned map[types.Object]bool
}

func (p receiverFields) check(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.CompositeLit:
		p.construction(node)
	case *ast.SelectorExpr:
		selection := p.pkg.TypesInfo.Selections[node]
		if selection != nil && p.owned[selection.Obj()] && !p.use(node) {
			delete(p.owned, selection.Obj())
		}
	}
	return true
}

func (p receiverFields) construction(literal *ast.CompositeLit) {
	shape, ok := underlying(p.pkg.TypesInfo.TypeOf(literal)).(*types.Struct)
	if !ok {
		return
	}
	for i, element := range literal.Elts {
		field := shape.Field(i)
		value := element
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			id, _ := pair.Key.(*ast.Ident)
			field, _ = p.pkg.TypesInfo.ObjectOf(id).(*types.Var)
			value = pair.Value
		}
		if p.owned[field] && !p.fresh(value) {
			delete(p.owned, field)
		}
	}
}

func (p receiverFields) fresh(value ast.Expr) bool {
	value = ast.Unparen(value)
	if p.pkg.TypesInfo.Types[value].IsNil() {
		return true
	}
	switch value := value.(type) {
	case *ast.CompositeLit:
		return true
	case *ast.CallExpr:
		return p.builtin(value) == "make"
	default:
		return false
	}
}

func (p receiverFields) builtin(call *ast.CallExpr) string {
	id, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok {
		return ""
	}
	object, ok := p.pkg.TypesInfo.Uses[id].(*types.Builtin)
	if !ok {
		return ""
	}
	return object.Name()
}

func (p receiverFields) use(selector *ast.SelectorExpr) bool {
	node, parent := p.parent(selector)
	switch parent := parent.(type) {
	case *ast.IndexExpr:
		return p.indexUse(node, parent)
	case *ast.RangeStmt:
		return parent.X == node
	case *ast.AssignStmt:
		return p.headerWrite(selector, node, parent)
	case *ast.CallExpr:
		return p.containerUse(selector, node, parent)
	default:
		return false
	}
}

func (p receiverFields) indexUse(node ast.Node, index *ast.IndexExpr) bool {
	return index.X == node && p.elementUse(index)
}

// An element value can be copied, but its inline storage cannot be exposed.
func (p receiverFields) elementUse(node ast.Node) bool {
	for parent := p.parents[node]; parent != nil; parent = p.parents[node] {
		if !p.elementContext(parent) {
			return false
		}
		node = parent
	}
	return true
}

func (p receiverFields) elementContext(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.UnaryExpr:
		return node.Op != token.AND
	case *ast.SliceExpr:
		return false
	case *ast.SelectorExpr:
		return !implicitReceiverAddress(p.pkg.TypesInfo, node)
	default:
		return true
	}
}

func (p receiverFields) headerWrite(selector *ast.SelectorExpr, node ast.Node, statement *ast.AssignStmt) bool {
	if statement.Tok != token.ASSIGN {
		return false
	}
	for i, place := range statement.Lhs {
		if place == node {
			return p.fresh(assignmentValue(statement.Rhs, i, len(statement.Lhs))) || p.selfAppend(selector, assignmentValue(statement.Rhs, i, len(statement.Lhs)))
		}
	}
	return false
}

func (p receiverFields) selfAppend(selector *ast.SelectorExpr, value ast.Expr) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || p.builtin(call) != "append" || len(call.Args) == 0 {
		return false
	}
	other, ok := ast.Unparen(call.Args[0]).(*ast.SelectorExpr)
	return ok && p.sameField(selector, other)
}

func (p receiverFields) sameField(left, right *ast.SelectorExpr) bool {
	a, aok := ast.Unparen(left.X).(*ast.Ident)
	b, bok := ast.Unparen(right.X).(*ast.Ident)
	if !aok || !bok {
		return false
	}
	return p.pkg.TypesInfo.ObjectOf(a) == p.pkg.TypesInfo.ObjectOf(b) && p.pkg.TypesInfo.ObjectOf(left.Sel) == p.pkg.TypesInfo.ObjectOf(right.Sel)
}

func (p receiverFields) containerUse(selector *ast.SelectorExpr, node ast.Node, call *ast.CallExpr) bool {
	switch p.builtin(call) {
	case "len", "cap", "delete", "clear":
		return len(call.Args) > 0 && call.Args[0] == node
	case "copy":
		return true
	case "append":
		return p.appendWrite(selector, call)
	default:
		return false
	}
}

func (p receiverFields) appendWrite(selector *ast.SelectorExpr, call *ast.CallExpr) bool {
	if len(call.Args) == 0 || ast.Unparen(call.Args[0]) != selector {
		return false
	}
	_, parent := p.parent(call)
	assignment, ok := parent.(*ast.AssignStmt)
	if !ok || len(assignment.Lhs) != 1 {
		return false
	}
	target, ok := ast.Unparen(assignment.Lhs[0]).(*ast.SelectorExpr)
	return ok && p.selfAppend(target, call)
}
