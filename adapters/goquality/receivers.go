package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Closed receiver types have only local constructions and direct method calls.
// This proves ownership of the outer slots, never their referenced storage.
type receiverProof struct {
	pkg     *packages.Package
	parents map[ast.Node]ast.Node
	defs    map[types.Object]*ast.Ident
}

func newReceiverProof(pkg *packages.Package) receiverProof {
	proof := receiverProof{pkg: pkg, parents: map[ast.Node]ast.Node{}, defs: map[types.Object]*ast.Ident{}}
	for _, file := range pkg.Syntax {
		indexParents(file, proof.parents)
	}
	for id, object := range pkg.TypesInfo.Defs {
		proof.defs[object] = id
	}
	return proof
}

func closedReceiverTypes(pkg *packages.Package) map[types.Object]bool {
	closed := map[types.Object]bool{}
	if opaqueGlobals(pkg) {
		return closed
	}
	proof := newReceiverProof(pkg)
	for object := range proof.defs {
		if privateReceiverType(object) {
			closed[object] = true
		}
	}
	proof.checkDefinitions(closed)
	for id, object := range pkg.TypesInfo.Uses {
		proof.checkUse(closed, id, object)
	}
	return closed
}

func privateReceiverType(object types.Object) bool {
	name, ok := object.(*types.TypeName)
	if !ok || name.Exported() || name.IsAlias() || !packageVariable(name) {
		return false
	}
	// A defined, non-alias package type has a named representation.
	named := name.Type().(*types.Named)
	_, record := named.Underlying().(*types.Struct)
	return record && named.TypeParams().Len() == 0
}

func receiverType(typ types.Type) types.Object {
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	if named, ok := typ.(*types.Named); ok {
		return named.Obj()
	}
	return nil
}

func (p receiverProof) checkDefinitions(closed map[types.Object]bool) {
	for object := range p.defs {
		variable, ok := object.(*types.Var)
		if ok && (packageVariable(variable) || variable.IsField()) {
			delete(closed, receiverType(variable.Type()))
		}
	}
}

func (p receiverProof) checkUse(closed map[types.Object]bool, id *ast.Ident, object types.Object) {
	name := receiverType(object.Type())
	if closed[name] && !p.closedUse(id, object, name) {
		delete(closed, name)
	}
}

func (p receiverProof) closedUse(id *ast.Ident, object types.Object, name types.Object) bool {
	switch object := object.(type) {
	case *types.TypeName:
		return p.typeUse(id)
	case *types.Var:
		return p.variableUse(id, object, name)
	default:
		return false
	}
}

func (p receiverProof) parent(node ast.Node) (ast.Node, ast.Node) {
	for {
		parent := p.parents[node]
		if _, parens := parent.(*ast.ParenExpr); !parens {
			return node, parent
		}
		node = parent
	}
}

func (p receiverProof) typeUse(id *ast.Ident) bool {
	_, parent := p.parent(id)
	if pointer, ok := parent.(*ast.StarExpr); ok {
		_, parent = p.parent(pointer)
	}
	switch parent := parent.(type) {
	case *ast.Field:
		return p.closedDeclaration(parent)
	case *ast.CompositeLit:
		return p.localConstruction(parent)
	case *ast.CallExpr:
		return p.newConstruction(parent)
	case *ast.ValueSpec:
		return p.localNames(parent.Names)
	default:
		return false
	}
}

func (p receiverProof) closedDeclaration(field *ast.Field) bool {
	list, _ := p.parents[field].(*ast.FieldList)
	switch owner := p.parents[list].(type) {
	case *ast.FuncDecl:
		return owner.Recv == list
	case *ast.FuncType:
		fn, ok := p.parents[owner].(*ast.FuncDecl)
		return ok && p.constructorResult(fn, list, field)
	default:
		return false
	}
}

func (p receiverProof) newConstruction(call *ast.CallExpr) bool {
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	builtin, ok := p.pkg.TypesInfo.Uses[id].(*types.Builtin)
	if !ok || builtin.Name() != "new" {
		return false
	}
	// Pointer holders can publish an allocated receiver through reflection.
	_, record := underlying(p.pkg.TypesInfo.TypeOf(call.Args[0])).(*types.Struct)
	return record && p.localConstruction(call)
}

func (p receiverProof) localNames(names []*ast.Ident) bool {
	for _, name := range names {
		object, ok := p.pkg.TypesInfo.Defs[name].(*types.Var)
		if !ok || packageVariable(object) {
			return false
		}
	}
	return len(names) != 0
}

func (p receiverProof) constructedNames(value ast.Expr, names []*ast.Ident) bool {
	for _, id := range names {
		if receiverType(p.pkg.TypesInfo.ObjectOf(id).Type()) != receiverType(p.pkg.TypesInfo.TypeOf(value)) {
			return false // An interface holder can publish the root indirectly.
		}
	}
	return p.localNames(names)
}

func (p receiverProof) localConstruction(value ast.Expr) bool {
	_, parent := p.parent(value)
	if address, ok := parent.(*ast.UnaryExpr); ok {
		// A typed record construction permits only address-taking here.
		_, parent = p.parent(address)
	}
	switch parent := parent.(type) {
	case *ast.AssignStmt:
		return p.localAssignment(parent)
	case *ast.ValueSpec:
		return p.constructedNames(value, parent.Names)
	case *ast.ReturnStmt:
		return p.constructorReturn(parent, receiverType(p.pkg.TypesInfo.TypeOf(value)))
	default:
		return false
	}
}

func (p receiverProof) localAssignment(statement *ast.AssignStmt) bool {
	if statement.Tok != token.DEFINE || len(statement.Lhs) != 1 {
		return false
	}
	id, ok := statement.Lhs[0].(*ast.Ident)
	return ok && p.localNames([]*ast.Ident{id})
}

func (p receiverProof) function(node ast.Node) ast.Node {
	for node != nil {
		switch node.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return node
		}
		node = p.parents[node]
	}
	return nil
}

func (p receiverProof) variableUse(id *ast.Ident, object types.Object, name types.Object) bool {
	if p.function(id) != p.function(p.defs[object]) {
		return false
	}
	node, parent := p.parent(id)
	selector, ok := parent.(*ast.SelectorExpr)
	if !ok {
		return p.constructorReturn(parent, name)
	}
	selection := p.pkg.TypesInfo.Selections[selector]
	if selection == nil || selector.X != node {
		return false
	}
	if selection.Kind() == types.FieldVal {
		return p.fieldUse(selector)
	}
	return p.methodUse(selector, selection, name)
}

func (p receiverProof) methodUse(selector *ast.SelectorExpr, selection *types.Selection, name types.Object) bool {
	signature := selection.Obj().Type().(*types.Signature)
	if len(selection.Index()) != 1 || receiverType(signature.Recv().Type()) != name {
		return false
	}
	node, parent := p.parent(selector)
	call, ok := parent.(*ast.CallExpr)
	if !ok || !p.receiverInvocation(call, node) {
		return false
	}
	switch p.parents[call].(type) {
	case *ast.GoStmt, *ast.DeferStmt:
		return false
	}
	return true
}

// Inspect finishes visiting before it returns and does not retain the method.
// The receiver must still pass every other ownership and escape check.
func (p receiverProof) receiverInvocation(call *ast.CallExpr, node ast.Node) bool {
	if call.Fun == node {
		return true
	}
	if len(call.Args) != 2 || call.Args[1] != node {
		return false
	}
	x := extractor{pkg: p.pkg}
	function, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
	return ok && resolvedName(function) == "go/ast.Inspect"
}

func (p receiverProof) fieldUse(selector *ast.SelectorExpr) bool {
	switch underlying(p.pkg.TypesInfo.TypeOf(selector)).(type) {
	case *types.Struct, *types.Array:
		return false // Inline aggregates can expose storage within the receiver.
	}
	_, parent := p.parent(selector)
	switch parent := parent.(type) {
	case *ast.UnaryExpr:
		return parent.Op != token.AND
	case *ast.SelectorExpr:
		return !implicitReceiverAddress(p.pkg.TypesInfo, parent)
	default:
		return true
	}
}

func implicitReceiverAddress(info *types.Info, selector *ast.SelectorExpr) bool {
	selection := info.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal {
		return false
	}
	signature := selection.Obj().Type().(*types.Signature)
	_, pointerMethod := underlying(signature.Recv().Type()).(*types.Pointer)
	_, pointerValue := underlying(info.TypeOf(selector.X)).(*types.Pointer)
	return pointerMethod && !pointerValue
}

func (x *extractor) localReceiverField(value *ast.SelectorExpr) bool {
	id, ok := ast.Unparen(value.X).(*ast.Ident)
	return x.closedReceiver && ok && x.pkg.TypesInfo.ObjectOf(id) == x.receiver
}

// This proves only the selected container's backing storage, not its elements.
func (x *extractor) localReceiverStorage(value *ast.SelectorExpr) bool {
	selection := x.pkg.TypesInfo.Selections[value]
	return x.localReceiverField(value) && selection != nil && len(selection.Index()) == 1 && x.receiverFields[selection.Obj()]
}
