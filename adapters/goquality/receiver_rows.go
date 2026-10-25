package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Row storage needs a separate lifetime proof. Empty private maps start with
// nil rows; only fresh slices and appends from those rows may replace them.
// Row aliases, publication and child access are outside this certificate.
func closedReceiverRows(pkg *packages.Package, fields map[types.Object]bool) map[types.Object]bool {
	owned := map[types.Object]bool{}
	for field := range fields {
		if sliceRowMap(field.Type()) {
			owned[field] = true
		}
	}
	if len(owned) == 0 {
		return owned
	}
	p := receiverRows{receiverFields{receiverProof: newReceiverProof(pkg), owned: owned}}
	for _, file := range pkg.Syntax {
		ast.Inspect(file, p.check)
	}
	return p.owned
}

type receiverRows struct{ receiverFields }

func sliceRowMap(typ types.Type) bool {
	shape, ok := underlying(typ).(*types.Map)
	if !ok {
		return false
	}
	_, ok = underlying(shape.Elem()).(*types.Slice)
	return ok
}

func (p receiverRows) check(node ast.Node) bool {
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

func (p receiverRows) construction(literal *ast.CompositeLit) {
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
		if p.owned[field] && !p.empty(value) {
			delete(p.owned, field)
		}
	}
}

func (p receiverRows) empty(value ast.Expr) bool {
	if literal, ok := ast.Unparen(value).(*ast.CompositeLit); ok {
		return len(literal.Elts) == 0
	}
	return p.fresh(value)
}

func (p receiverRows) use(field *ast.SelectorExpr) bool {
	node, parent := p.parent(field)
	switch parent := parent.(type) {
	case *ast.IndexExpr:
		return parent.X == node && p.rowUse(field, parent)
	case *ast.AssignStmt:
		return p.emptyWrite(node, parent)
	case *ast.CallExpr:
		return p.observation(node, parent)
	default:
		return false
	}
}

func (p receiverRows) emptyWrite(node ast.Node, statement *ast.AssignStmt) bool {
	return len(statement.Lhs) == 1 && len(statement.Rhs) == 1 && statement.Lhs[0] == node && p.empty(statement.Rhs[0])
}

func (p receiverRows) observation(node ast.Node, call *ast.CallExpr) bool {
	switch p.builtin(call) {
	case "len", "cap", "delete", "clear":
		return len(call.Args) > 0 && call.Args[0] == node
	default:
		return false
	}
}

func (p receiverRows) rowUse(field *ast.SelectorExpr, row *ast.IndexExpr) bool {
	node, parent := p.parent(row)
	switch parent := parent.(type) {
	case *ast.AssignStmt:
		return p.rowStore(field, node, parent)
	case *ast.CallExpr:
		return p.rowCall(field, node, parent)
	case *ast.BinaryExpr:
		return (&extractor{pkg: p.pkg}).nilComparison(parent)
	default:
		return false
	}
}

func (p receiverRows) rowStore(field *ast.SelectorExpr, node ast.Node, statement *ast.AssignStmt) bool {
	if statement.Tok != token.ASSIGN || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 || statement.Lhs[0] != node {
		return false
	}
	return p.fresh(statement.Rhs[0]) || p.rowAppend(field, statement.Rhs[0])
}

func (p receiverRows) rowAppend(field *ast.SelectorExpr, value ast.Expr) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || p.builtin(call) != "append" {
		return false
	}
	row, ok := ast.Unparen(call.Args[0]).(*ast.IndexExpr)
	if !ok {
		return false
	}
	return p.sameRowField(field, row)
}

func (p receiverRows) rowCall(field *ast.SelectorExpr, node ast.Node, call *ast.CallExpr) bool {
	if p.builtin(call) != "append" {
		return p.observation(node, call)
	}
	// Expanding a source slice copies its elements, not its backing store.
	if call.Args[0] != node {
		return call.Ellipsis.IsValid() && call.Args[len(call.Args)-1] == node
	}
	_, parent := p.parent(call)
	statement, ok := parent.(*ast.AssignStmt)
	if !ok {
		return false
	}
	return p.appendDestination(field, statement)
}

func (p receiverRows) appendDestination(field *ast.SelectorExpr, statement *ast.AssignStmt) bool {
	if len(statement.Lhs) != 1 {
		return false
	}
	row, ok := ast.Unparen(statement.Lhs[0]).(*ast.IndexExpr)
	return ok && p.rowStore(field, row, statement) && p.sameRowField(field, row)
}

func (p receiverRows) sameRowField(field *ast.SelectorExpr, row *ast.IndexExpr) bool {
	other, ok := ast.Unparen(row.X).(*ast.SelectorExpr)
	return ok && p.sameField(field, other)
}

func (x *extractor) localReceiverRow(row *ast.IndexExpr) bool {
	field, ok := ast.Unparen(row.X).(*ast.SelectorExpr)
	if !ok || !x.localReceiverField(field) {
		return false
	}
	selection := x.pkg.TypesInfo.Selections[field]
	return selection != nil && len(selection.Index()) == 1 && x.receiverRows[selection.Obj()]
}
