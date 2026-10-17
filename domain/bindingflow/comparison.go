package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/constant"
	"go/token"
)

func (e *evaluation) comparison(ref pdg.Ref, operator string) Value {
	kind := comparisonToken(operator)
	if kind == token.ILLEGAL {
		return Value{}
	}
	left, right := e.value(e.query.Operand(ref, 1)), e.value(e.query.Operand(ref, 2))
	if truth, known := nilComparison(kind, left, right); known {
		return Value{Scalar: constant.MakeBool(truth)}
	}
	if !comparableScalars(kind, left.Scalar, right.Scalar) {
		return Value{}
	}
	return Value{Scalar: constant.MakeBool(constant.Compare(left.Scalar, kind, right.Scalar))}
}

// NilPredicate identifies the actual nil-comparison subject and true outcome.
// Transparent wrappers use canonical operands; absent/cyclic edges stay unknown.
func (q *Query) NilPredicate(ref pdg.Ref, facts Facts) (pdg.Ref, bool) {
	if q == nil {
		return 0, false
	}
	negated := false
	for range q.graph.Nodes {
		if ref == 0 || int(ref) > len(q.graph.Nodes) {
			return 0, false
		}
		next, invert := q.predicateNext(ref)
		if next != 0 {
			ref = next
			negated = negated != invert
			continue
		}
		attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
		subject, notNil := q.nilSubject(ref, q.operator(attrs), negated, facts)
		return q.IdentitySource(subject, facts), notNil
	}
	return 0, false
}

// IdentitySource follows only transparent value wrappers with explicit inputs.
// It supplies no storage/alias equivalence for the terminal value.
func (q *Query) IdentitySource(ref pdg.Ref, facts Facts) pdg.Ref {
	if q == nil {
		return 0
	}
	for range q.graph.Nodes {
		if ref == 0 || int(ref) > len(q.graph.Nodes) {
			return 0
		}
		next, wrapped := q.identityOperand(ref, facts)
		if !wrapped {
			return ref
		}
		ref = next
	}
	return 0
}

func (q *Query) nilSubject(ref pdg.Ref, operator string, negated bool, facts Facts) (pdg.Ref, bool) {
	if operator != "==" && operator != "!=" {
		return 0, false
	}
	left, right, known := q.operandPair(ref)
	if !known {
		return 0, false
	}
	notNil := (operator == "!=") != negated
	if facts.NilConstant(right) {
		return left, notNil
	}
	if facts.NilConstant(left) {
		return right, notNil
	}
	return 0, false
}

func (q *Query) predicateNext(ref pdg.Ref) (pdg.Ref, bool) {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	if next := q.truthOperand(ref, attrs); next != 0 {
		return next, false
	}
	switch q.operator(attrs) {
	case "!":
		return q.Operand(ref, 1), true
	case "parentheses":
		return q.Operand(ref, 1), false
	default:
		return 0, false
	}
}

func (q *Query) operandPair(ref pdg.Ref) (pdg.Ref, pdg.Ref, bool) {
	left, right := q.Operand(ref, 1), q.Operand(ref, 2)
	return left, right, left != 0 && right != 0
}

func (q *Query) identityOperand(ref pdg.Ref, facts Facts) (pdg.Ref, bool) {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	switch q.operator(attrs) {
	case "parentheses":
		return q.Operand(ref, 1), true
	case "conversion":
		if facts.Nodes[ref].Conversion != IdentityConversion {
			return 0, true
		}
		return q.Operand(ref, 1), true
	default:
		return 0, false
	}
}

func comparisonToken(operator string) token.Token {
	switch operator {
	case "==", "!=":
		return equalityToken(operator)
	case "<":
		return token.LSS
	case "<=":
		return token.LEQ
	case ">":
		return token.GTR
	case ">=":
		return token.GEQ
	default:
		return token.ILLEGAL
	}
}

func nilComparison(operator token.Token, left, right Value) (bool, bool) {
	if !equalityOperator(operator) {
		return false, false
	}
	if left.Nil {
		return compareNil(operator, right)
	}
	if right.Nil {
		return compareNil(operator, left)
	}
	return false, false
}

func compareNil(operator token.Token, other Value) (bool, bool) {
	if other.Nil {
		return operator == token.EQL, true
	}
	if other.NonNil {
		return operator == token.NEQ, true
	}
	return false, false
}

func equalityToken(operator string) token.Token {
	if operator == "==" {
		return token.EQL
	}
	return token.NEQ
}

func equalityOperator(operator token.Token) bool {
	return operator == token.EQL || operator == token.NEQ
}

func comparableScalars(operator token.Token, left, right constant.Value) bool {
	if left == nil || right == nil {
		return false
	}
	return scalarKindsComparable(operator, left.Kind(), right.Kind())
}

func scalarKindsComparable(operator token.Token, l, r constant.Kind) bool {
	if numericKind(l) && numericKind(r) {
		return numericComparison(operator, l, r)
	}
	if l != r {
		return false
	}
	switch l {
	case constant.String:
		return true
	case constant.Bool:
		return equalityOperator(operator)
	default:
		return false
	}
}

func numericKind(kind constant.Kind) bool {
	switch kind {
	case constant.Int, constant.Float, constant.Complex:
		return true
	default:
		return false
	}
}

func numericComparison(operator token.Token, left, right constant.Kind) bool {
	if operator == token.EQL || operator == token.NEQ {
		return true
	}
	return left != constant.Complex && right != constant.Complex
}
