package goquality

import (
	"go/ast"
	"go/types"
)

// A map may be read without retaining its outer storage. Child values must
// have a bounded shape that cannot contain an alias back to that outer map.
type mapParameters struct {
	receiverProof
	functions map[types.Object]*ast.FuncDecl
	uses      map[types.Object][]*ast.Ident
	states    map[types.Object]uint8
}

func newMapParameters(proof receiverProof) *mapParameters {
	p := &mapParameters{receiverProof: proof, functions: map[types.Object]*ast.FuncDecl{}, uses: map[types.Object][]*ast.Ident{}, states: map[types.Object]uint8{}}
	for object, id := range proof.defs {
		if fn, ok := proof.parents[id].(*ast.FuncDecl); ok && mapHelper(fn, object.Type().(*types.Signature)) {
			p.functions[object] = fn
		}
	}
	for id, object := range proof.pkg.TypesInfo.Uses {
		p.uses[object] = append(p.uses[object], id)
	}
	return p
}

func mapHelper(fn *ast.FuncDecl, signature *types.Signature) bool {
	return fn.Body != nil && fn.Recv == nil && fn.Type.TypeParams == nil && !signature.Variadic()
}

func (p *mapParameters) argument(call *ast.CallExpr, value ast.Node) bool {
	if !p.synchronous(call) || call.Ellipsis.IsValid() {
		return false
	}
	object := (&extractor{pkg: p.pkg}).calledObject(unwrappedCallee(call.Fun))
	fn := p.functions[object]
	if fn == nil {
		return false
	}
	signature := object.Type().(*types.Signature)
	for index, argument := range call.Args {
		if argument == value {
			return p.parameter(fn, signature.Params().At(index))
		}
	}
	return false
}

// In-progress parameters reject cycles; no recursive purity assumption proves
// that a map cannot be retained by a helper.
func (p *mapParameters) parameter(fn *ast.FuncDecl, parameter *types.Var) bool {
	if state := p.states[parameter]; state != 0 {
		return state == 3
	}
	p.states[parameter] = 1
	if !boundedMap(parameter.Type()) {
		p.states[parameter] = 2
		return false
	}
	for _, use := range p.uses[parameter] {
		if p.function(use) != fn || !p.use(use) {
			p.states[parameter] = 2
			return false
		}
	}
	p.states[parameter] = 3
	return true
}

func boundedMap(typ types.Type) bool {
	shape, ok := underlying(typ).(*types.Map)
	if !ok {
		return false
	}
	return mapChild(shape.Key(), 1) && mapChild(shape.Elem(), 3)
}

func mapChild(typ types.Type, depth int) bool {
	if depth == 0 {
		return false
	}
	switch shape := underlying(typ).(type) {
	case *types.Basic:
		return shape.Kind() != types.UnsafePointer
	case *types.Array:
		return mapChild(shape.Elem(), depth-1)
	case *types.Slice:
		return mapChild(shape.Elem(), depth-1)
	default:
		return false
	}
}

func (p *mapParameters) use(id *ast.Ident) bool {
	if !p.synchronous(id) {
		return false
	}
	node, parent := p.parent(id)
	switch parent := parent.(type) {
	case *ast.IndexExpr:
		return parent.X == node
	case *ast.RangeStmt:
		return parent.X == node
	case *ast.CallExpr:
		return p.callUse(parent, node)
	default:
		return false
	}
}

func (p *mapParameters) callUse(call *ast.CallExpr, node ast.Node) bool {
	if builtin, ok := (&extractor{pkg: p.pkg}).calledObject(unwrappedCallee(call.Fun)).(*types.Builtin); ok {
		return builtin.Name() == "len" && len(call.Args) == 1 && call.Args[0] == node
	}
	return p.argument(call, node)
}

func (p *mapParameters) synchronous(node ast.Node) bool {
	for parent := p.parents[node]; parent != nil; parent = p.parents[parent] {
		switch parent.(type) {
		case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
			return false
		}
	}
	return true
}
