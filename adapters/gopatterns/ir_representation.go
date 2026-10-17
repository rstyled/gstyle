package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

type representationInput struct {
	Position pdg.Position `json:"position"`
	Kind     string       `json:"kind"`
}

func (b *irBuilder) representationFacts(fn *ast.FuncDecl) {
	signature := b.info.Defs[fn.Name].Type().(*types.Signature)
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		b.representationNode(node, signature)
		return true
	})
}

func (b *irBuilder) representationNode(node ast.Node, signature *types.Signature) {
	targets, values := b.representationOperands(node, signature)
	if targets == nil {
		return
	}
	ref := b.nodes[node]
	if ref == 0 {
		return
	}
	facts := make([]representationInput, len(targets))
	for i, target := range targets {
		facts[i] = representationInput{Position: pdg.Position(i + 1), Kind: representationKind(b.representationSource(values, i, target), target)}
	}
	data, _ := json.Marshal(facts)
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:input-representation", string(data))...)
}

// representationOperands obtains compiler source/target types before publishing
// the transfer inventory; it does not follow values or source control flow.
func (b *irBuilder) representationOperands(node ast.Node, signature *types.Signature) ([]types.Type, []ast.Expr) {
	switch node := node.(type) {
	case *ast.CallExpr:
		return b.callRepresentations(node)
	case *ast.AssignStmt:
		return b.representationTargets(node.Lhs), node.Rhs
	case *ast.ValueSpec:
		return b.representationTargets(memoryDeclaredTargets(node.Names)), node.Values
	case *ast.ReturnStmt:
		return representationResults(signature), node.Results
	default:
		return nil, nil
	}
}

func (b *irBuilder) representationTargets(expressions []ast.Expr) []types.Type {
	targets := make([]types.Type, len(expressions))
	for i, expression := range expressions {
		targets[i] = b.info.TypeOf(expression)
	}
	return targets
}

func (b *irBuilder) representationSource(values []ast.Expr, index int, zero types.Type) types.Type {
	if len(values) == 0 {
		return zero
	} // Zero initialization or a bare named return.
	if len(values) == 1 {
		if tuple, ok := b.info.TypeOf(values[0]).(*types.Tuple); ok {
			return representationTupleType(tuple, index)
		}
	}
	if index >= len(values) {
		return nil
	}
	return b.info.TypeOf(values[index])
}

func representationResults(signature *types.Signature) []types.Type {
	targets := make([]types.Type, signature.Results().Len())
	for i := range targets {
		targets[i] = signature.Results().At(i).Type()
	}
	return targets
}

func representationTupleType(tuple *types.Tuple, index int) types.Type {
	if index >= tuple.Len() {
		return nil
	}
	return tuple.At(index).Type()
}

func representationKind(source, target types.Type) string {
	if source == nil || target == nil {
		return "unknown"
	}
	if types.Identical(source, target) || untypedNilRepresentation(source) {
		return "identity"
	}
	return convertedRepresentation(source, target)
}

func convertedRepresentation(source, target types.Type) string {
	if genericRepresentation(source) || genericRepresentation(target) {
		return "unknown"
	}
	if interfaceType(target) && !interfaceType(source) {
		return "boxed"
	}
	return "identity"
}

func untypedNilRepresentation(typ types.Type) bool {
	basic, ok := typ.(*types.Basic)
	return ok && basic.Kind() == types.UntypedNil
}

func genericRepresentation(typ types.Type) bool {
	_, generic := types.Unalias(typ).(*types.TypeParam)
	return generic
}
