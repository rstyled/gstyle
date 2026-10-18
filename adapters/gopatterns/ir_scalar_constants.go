package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/types"
)

func (b *irBuilder) scalarConstant(id *ast.Ident) {
	constant, ok := b.info.Uses[id].(*types.Const)
	if !ok {
		return
	}
	b.addNode(id, pdg.Computation, pdg.Attributes{Literal: b.pool.Known(constant.Val().ExactString()), Operator: b.pool.Known("literal")})
}

func (b *irBuilder) scalarInterface(fn *ast.FuncDecl) bool {
	return fn.Type.TypeParams == nil && b.scalarFields(fn.Recv) && b.scalarFields(fn.Type.Params) && b.scalarFields(fn.Type.Results)
}

func (b *irBuilder) scalarFields(fields *ast.FieldList) bool {
	if fields == nil {
		return true
	}
	for _, field := range fields.List {
		typ := b.info.TypeOf(field.Type)
		if typ == nil {
			return false
		}
		basic, ok := typ.Underlying().(*types.Basic)
		if !ok || basic.Info()&(types.IsInteger|types.IsBoolean) == 0 {
			return false
		}
	}
	return true
}
