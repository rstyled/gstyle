package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/types"
)

// Compiler leaf facts describe values. Operand edges describe their inputs.
func (b *irBuilder) canonicalValueFacts(fn *ast.FuncDecl) {
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		ref := b.nodes[node]
		if ref == 0 {
			return true
		}
		fact := b.canonicalValueFact(expression)
		if fact == "" {
			return true
		}
		attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
		attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:value-fact", fact)...)
		return true
	})
}
func (b *irBuilder) canonicalValueFact(expr ast.Expr) string {
	if fact := canonicalConstantFact(b.info.Types[expr]); fact != "" {
		return fact
	}
	if b.canonicalNonNilValue(expr) {
		return `{"non_nil":true}`
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || !b.info.Types[call.Fun].IsType() {
		return ""
	}
	return `{"conversion":"` + b.canonicalConversion(call) + `"}`
}
func (b *irBuilder) canonicalConversion(call *ast.CallExpr) string {
	source := b.info.TypeOf(call.Args[0])
	target := b.info.TypeOf(call)
	if b.info.Types[call.Args[0]].IsNil() {
		return "nil"
	}
	if types.Identical(source, target) || booleanConversion(source, target) || callableConversion(source, target) {
		return "identity"
	}
	return conversionNilness(memoryTypeKind(source), memoryTypeKind(target))
}
func conversionNilness(source, target string) string {
	if source == "unknown" {
		return "unknown"
	}
	if target == "interface" && source != "interface" {
		return "boxed"
	}
	if source != target {
		return "unknown"
	}
	switch source {
	case "pointer", "slice", "map", "channel", "function", "interface":
		return "nilness"
	default:
		return "unknown"
	}
}

func canonicalConstantFact(value types.TypeAndValue) string {
	if value.IsNil() {
		return `{"nil":true}`
	}
	if value.Value != nil && value.Value.Kind() == constant.Bool {
		return `{"boolean":` + value.Value.ExactString() + `}`
	}
	return ""
}

// Boolean conversions preserve the complete two-value domain, including named
// types. Other scalar conversions can change values and keep their old contract.
func booleanConversion(source, target types.Type) bool {
	from, ok := source.Underlying().(*types.Basic)
	if !ok || from.Info()&types.IsBoolean == 0 {
		return false
	}
	to, ok := target.Underlying().(*types.Basic)
	return ok && to.Info()&types.IsBoolean != 0
}

// Function type conversions preserve the selected code and captured cells.
func callableConversion(source, target types.Type) bool {
	_, from := source.Underlying().(*types.Signature)
	_, to := target.Underlying().(*types.Signature)
	return from && to
}
