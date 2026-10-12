package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (b *irBuilder) interfaces(fn *ast.FuncDecl) {
	b.fields(fn.Recv, pdg.ReceiverRole)
	b.fields(fn.Type.Params, pdg.ParameterRole)
	b.fields(fn.Type.Results, pdg.ReturnRole)
}

func (b *irBuilder) fields(fields *ast.FieldList, role pdg.Role) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		b.field(field, role)
	}
}

func (b *irBuilder) field(field *ast.Field, role pdg.Role) {
	if len(field.Names) == 0 {
		b.parameter(field, nil, role)
		return
	}
	for _, name := range field.Names {
		b.parameter(field, name, role)
	}
}

// parameter keeps interface positions and formal operation nodes in one step.
func (b *irBuilder) parameter(field *ast.Field, name *ast.Ident, role pdg.Role) {
	output := role == pdg.ReturnRole
	category := pdg.FormalInput
	position := len(b.graph.Function.Inputs)
	if output {
		category = pdg.FormalOutput
		position = len(b.graph.Function.Outputs)
	}
	param := b.parameterFacts(field, name, role, uint32(position))
	if output {
		b.graph.Function.Outputs = append(b.graph.Function.Outputs, param)
	} else {
		b.graph.Function.Inputs = append(b.graph.Function.Inputs, param)
	}
	var symbols []pdg.Ref
	if param.Symbol != 0 {
		symbols = []pdg.Ref{param.Symbol}
	}
	attrs := pdg.Attributes{Position: pdg.Position(param.Position + 1), Symbols: symbols}
	node := b.appendNode(field, category, attrs)
	if output {
		b.outputs = append(b.outputs, node)
	}
	b.definition(node, param.Symbol)
}

func (b *irBuilder) parameterFacts(field *ast.Field, name *ast.Ident, role pdg.Role, position uint32) pdg.Parameter {
	_, variadic := field.Type.(*ast.Ellipsis)
	p := pdg.Parameter{
		Position: position, Role: role, Name: b.pool.Known(""), Type: b.typeFact(field.Type),
		Variadic: pdg.BooleanFact{Status: pdg.Known, Value: variadic},
	}
	if name != nil {
		p.Name = b.pool.Known(name.Name)
		p.Symbol = b.symbol(name, symbolRole(role))
	}
	return p
}

func symbolRole(role pdg.Role) pdg.Role {
	if role == pdg.ReturnRole {
		return pdg.OutputRole
	}
	return role
}

func (b *irBuilder) typeFact(expr ast.Expr) pdg.Fact {
	typ := b.info.TypeOf(expr)
	if typ == nil {
		return b.pool.Missing(pdg.Unknown, "Type is unresolved.")
	}
	return b.pool.Known(irTypeString(typ))
}

func irTypeString(typ types.Type) string {
	return types.TypeString(typ, func(pkg *types.Package) string { return pkg.Path() })
}
