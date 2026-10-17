package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Compiler selection indexes describe fields, including promoted fields. Input
// positions describe required runtime values, not a second dependency graph.
func (b *irBuilder) dependencySourceFacts(expression ast.Expr, attrs *pdg.Attributes) {
	positions := b.expressionInputs(expression)
	if b.pool.Tables.Text(attrs.Operator.Value) == "function-value" {
		positions = nil
	}
	if positions == nil {
		positions = []pdg.Position{}
	}
	data, _ := json.Marshal(positions) // Numeric slices always encode.
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:value-operands", string(data))...)
	b.fieldPathFact(expression, attrs)
	b.tupleArityFact(expression, attrs)
}
func (b *irBuilder) fieldPathFact(expression ast.Expr, attrs *pdg.Attributes) {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return
	}
	selection := b.info.Selections[selector]
	if selection == nil || selection.Kind() != types.FieldVal {
		return
	}
	data, _ := json.Marshal(selection.Index())
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:field-path", string(data))...)
}
func (b *irBuilder) expressionInputs(expression ast.Expr) []pdg.Position {
	switch expression := expression.(type) {
	case *ast.BinaryExpr:
		return []pdg.Position{1, 2}
	case *ast.UnaryExpr, *ast.ParenExpr, *ast.StarExpr, *ast.TypeAssertExpr:
		return []pdg.Position{1}
	case *ast.CompositeLit:
		return b.valuePositions(expression.Elts, 0)
	case *ast.KeyValueExpr:
		return b.keyValueInputs(expression)
	default:
		return b.accessInputs(expression)
	}
}
func (b *irBuilder) accessInputs(expression ast.Expr) []pdg.Position {
	switch expression := expression.(type) {
	case *ast.SelectorExpr:
		if selection := b.info.Selections[expression]; selection != nil {
			return []pdg.Position{1}
		}
		return nil
	case *ast.IndexExpr:
		return b.valuePositions([]ast.Expr{expression.X, expression.Index}, 0)
	case *ast.SliceExpr:
		return b.valuePositions([]ast.Expr{expression.X, expression.Low, expression.High, expression.Max}, 0)
	case *ast.CallExpr:
		return b.callInputs(expression)
	default:
		return nil
	}
}
func (b *irBuilder) callInputs(call *ast.CallExpr) []pdg.Position {
	if !b.implicitReceiver(call.Fun) {
		return b.valuePositions(call.Args, 0)
	}
	return append([]pdg.Position{1}, b.valuePositions(call.Args, 1)...)
}
func (b *irBuilder) valuePositions(expressions []ast.Expr, offset int) []pdg.Position {
	var positions []pdg.Position
	for i, expression := range expressions {
		if expression == nil || b.info.Types[expression].IsType() {
			continue
		}
		positions = append(positions, pdg.Position(i+offset+1))
	}
	return positions
}

func (b *irBuilder) keyValueInputs(expression *ast.KeyValueExpr) []pdg.Position {
	if name, ok := expression.Key.(*ast.Ident); ok && literalFieldIdentifier(name, b.info) {
		return []pdg.Position{2}
	}
	return []pdg.Position{1, 2}
}
