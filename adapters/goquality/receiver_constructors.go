package goquality

import (
	"go/ast"
	"go/types"
)

// A private constructor can transfer a root to local callers without publishing
// it. All other uses of that root still pass the closed-receiver checks.
func (p receiverProof) constructorResult(fn *ast.FuncDecl, list *ast.FieldList, field *ast.Field) bool {
	if fn.Type.Results != list || len(list.List) != 1 || len(field.Names) != 0 {
		return false
	}
	return p.privateConstructor(fn)
}

func (p receiverProof) constructorReturn(node ast.Node, name types.Object) bool {
	if _, ok := node.(*ast.ReturnStmt); !ok {
		return false
	}
	fn, ok := p.function(node).(*ast.FuncDecl)
	if !ok {
		return false
	}
	signature := p.pkg.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	if signature.Results().Len() != 1 || receiverType(signature.Results().At(0).Type()) != name {
		return false
	}
	return p.privateConstructor(fn)
}

func (p receiverProof) privateConstructor(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Body == nil || fn.Name.IsExported() || fn.Type.TypeParams != nil {
		return false
	}
	return p.constructorCallers(p.pkg.TypesInfo.Defs[fn.Name])
}

func (p receiverProof) constructorCallers(function types.Object) bool {
	for id, object := range p.pkg.TypesInfo.Uses {
		if object != function {
			continue
		}
		if !p.localConstructorCall(id) {
			return false
		}
	}
	return true
}

func (p receiverProof) localConstructorCall(id *ast.Ident) bool {
	node, parent := p.parent(id)
	call, ok := parent.(*ast.CallExpr)
	if !ok || call.Fun != node {
		return false
	}
	_, holder := p.parent(call)
	switch holder.(type) {
	case *ast.AssignStmt, *ast.ValueSpec:
		return p.localConstruction(call)
	default:
		return false
	}
}
