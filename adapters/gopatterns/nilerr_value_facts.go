package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Only compiler constants, types and fixed literal shapes enter as leaf facts.
// Expression dependency selection belongs to the canonical operand query.
func (q *nilErrGraph) indexValueFacts(info *types.Info) error {
	facts, err := q.copies.ValueFacts()
	if err != nil {
		return err
	}
	q.valueFacts = facts
	q.valueFacts.BooleanSymbols = map[pdg.Ref]bool{}
	q.valueLocations = map[bindingflow.PathKey][]pdg.Ref{}
	for ref, node := range q.source {
		expression, ok := node.(ast.Expr)
		if !ok {
			continue
		}
		q.indexExpressionFact(ref, expression, info)
	}
	q.indexBooleanSymbols()
	q.indexStoragePaths()
	return nil
}

func (q *nilErrGraph) nilCheck(ref pdg.Ref, info *types.Info) *nilErrCheck {
	subject, notNil := q.copies.NilPredicate(ref, q.valueFacts)
	if !q.errorRead(subject, info) {
		return nil
	}
	path := q.copies.Path(subject, q.valueFacts)
	if path.Root == 0 {
		return nil
	}
	return &nilErrCheck{path: path, isNotNil: notNil}
}

func (q *nilErrGraph) indexExpressionFact(ref pdg.Ref, expression ast.Expr, info *types.Info) {
	value, known := nilErrLeafValue(expression, info)
	if !known {
		return
	}
	fact := q.valueFacts.Nodes[ref]
	if fact.Known && !value.Empty {
		return
	}
	// Empty literal shape and whole-value non-nilness are independent facets.
	q.valueFacts.Nodes[ref] = bindingflow.Fact{Known: true, Conversion: fact.Conversion, Value: bindingflow.Value{Scalar: value.Scalar, Nil: value.Nil, Empty: value.Empty, NonNil: fact.Value.NonNil}}

}

func (q *nilErrGraph) indexBooleanSymbols() {
	for symbol, object := range q.objects {
		basic, ok := object.Type().Underlying().(*types.Basic)
		q.valueFacts.BooleanSymbols[symbol] = ok && basic.Kind() == types.Bool
	}
}

func (q *nilErrGraph) errorRead(subject pdg.Ref, info *types.Info) bool {
	if subject == 0 {
		return false
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[subject-1])
	if q.graph.Nodes[subject-1].Category != pdg.Read || len(attrs.Reads) == 0 {
		return false
	}
	expression, ok := q.source[subject].(ast.Expr)
	return ok && isErrorExpr(info, expression)
}

// All conversion facts must exist before a base path is resolved.
func (q *nilErrGraph) indexStoragePaths() {
	for i, node := range q.graph.Nodes {
		attrs := q.graph.NodeAttributes(node)
		if node.Category != pdg.Read || len(attrs.Reads) == 0 {
			continue
		}
		ref := pdg.Ref(i + 1)
		path := q.copies.Path(ref, q.valueFacts)
		if path.Root == 0 {
			continue
		}
		key := path.Key()
		q.valueLocations[key] = append(q.valueLocations[key], ref)
	}
}

func (e nilErrEvidence) evaluation() bindingflow.Evaluation {
	return bindingflow.Evaluation{Bindings: e.values, Truths: e.truths, Nodes: e.guardNodeValues(), Facts: e.graph.valueFacts}
}

// Canonical field paths and actual base operands identify guarded reads.
func (e nilErrEvidence) guardNodeValues() map[pdg.Ref]bindingflow.Value {
	values := map[pdg.Ref]bindingflow.Value{}
	for _, guard := range e.guards {
		for _, ref := range e.graph.valueLocations[guard.path.Key()] {
			values[ref] = bindingflow.Value{Nil: !guard.isNotNil, NonNil: guard.isNotNil}
		}
	}
	return values
}

func nilErrLeafValue(expression ast.Expr, info *types.Info) (bindingflow.Value, bool) {
	typed := info.Types[expression]
	if typed.IsNil() {
		return bindingflow.Value{Nil: true}, true
	}
	if typed.Value != nil {
		return bindingflow.Value{Scalar: typed.Value}, true
	}
	if literal, ok := expression.(*ast.CompositeLit); ok && emptyNilErrLiteral(literal, info) {
		return bindingflow.Value{Empty: true}, true
	}
	return bindingflow.Value{}, false
}
