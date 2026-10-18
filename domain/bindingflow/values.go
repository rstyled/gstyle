package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/constant"
)

// Value contains scalar facts, a symbolic Boolean or an immutable callable set.
// Unknown is the zero value. Nil and NonNil describe whole-value nilness.
type Value struct {
	Callables          CallableSet
	Scalar             constant.Value
	Boolean            pdg.Ref
	Negated            bool
	Nil, NonNil, Empty bool
}

func (v Value) Zero() bool {
	if v.Nil || v.Empty {
		return true
	}
	return scalarZero(v.Scalar)
}

func (v Value) Bool() (bool, bool) {
	if v.Scalar == nil || v.Scalar.Kind() != constant.Bool {
		return false, false
	}
	return constant.BoolVal(v.Scalar), true
}

func (value Value) resolveTruth(truths map[pdg.Ref]bool) Value {
	if value.Boolean == 0 {
		return value
	}
	if truth, known := truths[value.Boolean]; known {
		return Value{Scalar: constant.MakeBool(truth != value.Negated)}
	}
	return value
}

type Conversion uint8

const (
	UnknownConversion Conversion = iota
	IdentityConversion
	NilConversion
	BoxedConversion
	NilnessConversion
)

// Fact is one compiler-owned node record. Known marks an exact value; a
// conversion rule instead transfers the actual canonical operand.
type Fact struct {
	Value      Value
	Known      bool
	Conversion Conversion
}

// Facts contain no AST or source dependency graph.
type Facts struct {
	SingleCellSymbols map[pdg.Ref]bool
	CapturedSymbols   map[pdg.Ref]bool
	Nodes             map[pdg.Ref]Fact
	BooleanSymbols    map[pdg.Ref]bool
}

func (facts Facts) NilConstant(ref pdg.Ref) bool {
	fact := facts.Nodes[ref]
	return ref != 0 && fact.Known && fact.Value.Nil
}

type Evaluation struct {
	// Captures holds call-time cell facts: current values for proved single
	// cells, otherwise family invariants. Deferred invocation supplies live cells.
	Captures map[pdg.Ref]Value
	Results  map[pdg.Ref][]Value
	Bindings map[pdg.Ref]Value
	Truths   map[pdg.Ref]bool
	Nodes    map[pdg.Ref]Value
	Facts    Facts
}

type evaluation struct {
	query   *Query
	context Evaluation
	cache   map[pdg.Ref]evaluatedValue
}

func (e *evaluation) value(ref pdg.Ref) Value {
	if ref == 0 || int(ref) > len(e.query.graph.Nodes) {
		return Value{}
	}
	cached := e.cache[ref]
	switch cached.status {
	case 1:
		return Value{} // Cyclic value expressions cannot invent an exact fact.
	case 2:
		return cached.value
	}
	e.cache[ref] = evaluatedValue{status: 1}
	value := e.compute(ref)
	e.cache[ref] = evaluatedValue{status: 2, value: value}
	return value
}

func (e *evaluation) compute(ref pdg.Ref) Value {
	if fact := e.context.Facts.Nodes[ref]; fact.Known {
		return fact.Value
	}
	if value, known := e.context.Nodes[ref]; known {
		return value
	}
	q := e.query
	node := q.graph.Nodes[ref-1]
	attrs := q.graph.NodeAttributes(node)
	if subject := q.truthOperand(ref, attrs); subject != 0 {
		return e.value(subject)
	}
	if node.Category == pdg.Read {
		return e.read(attrs)
	}
	return e.operation(ref, q.operator(attrs))
}

func (e *evaluation) read(attrs pdg.Attributes) Value {
	if len(attrs.Reads) != 1 || !directBinding(attrs.Reads[0]) {
		return Value{}
	}
	symbol := attrs.Reads[0].Symbol
	value, known := e.context.Bindings[symbol]
	if !known && e.context.Facts.BooleanSymbols[symbol] {
		value.Boolean = symbol
	}
	return value.resolveTruth(e.context.Truths)
}

func (e *evaluation) operation(ref pdg.Ref, operator string) Value {
	switch operator {
	case "parentheses":
		return e.value(e.query.Operand(ref, 1))
	case "!":
		return negateValue(e.value(e.query.Operand(ref, 1)))
	case "conversion":
		return e.conversion(ref)
	case "&&", "||":
		return e.logical(ref, operator)
	default:
		return e.comparison(ref, operator)
	}
}

func (e *evaluation) conversion(ref pdg.Ref) Value {
	source := e.query.Operand(ref, 1)
	switch e.context.Facts.Nodes[ref].Conversion {
	case IdentityConversion:
		return e.value(source)
	case BoxedConversion:
		if source != 0 {
			return Value{NonNil: true}
		}
	case NilnessConversion:
		value := e.value(source)
		return Value{Nil: value.Nil, NonNil: value.NonNil}
	case NilConversion:
		return Value{Nil: e.context.Facts.NilConstant(source)}
	}
	return Value{}
}

func (e *evaluation) logical(ref pdg.Ref, operator string) Value {
	left, lk := e.value(e.query.Operand(ref, 1)).Bool()
	if truth, known := logicalShortCircuit(operator, left, lk); known {
		return Value{Scalar: constant.MakeBool(truth)}
	}
	right, rk := e.value(e.query.Operand(ref, 2)).Bool()
	if truth, known := logicalShortCircuit(operator, right, rk); known {
		return Value{Scalar: constant.MakeBool(truth)}
	}
	if !lk || !rk {
		return Value{}
	}
	return Value{Scalar: constant.MakeBool(left && right)} // Both non-short-circuit operands agree.
}

type evaluatedValue struct {
	status uint8
	value  Value
}

func (q *Query) Evaluate(ref pdg.Ref, context Evaluation) Value {
	if q == nil {
		return Value{}
	}
	run := evaluation{query: q, context: context, cache: map[pdg.Ref]evaluatedValue{}}
	return run.value(ref)
}

func (q *Query) truthOperand(ref pdg.Ref, attrs pdg.Attributes) pdg.Ref {
	switch q.graph.Tables.Text(attrs.Construct) {
	case "if", "for", "short-circuit":
		return q.Operand(ref, 1)
	default:
		if q.operator(attrs) == "truth-test" {
			return q.Operand(ref, 1)
		}
		return 0
	}
}

func (q *Query) operator(attrs pdg.Attributes) string {
	if attrs.Operator.Status != pdg.Known {
		return ""
	}
	return q.graph.Tables.Text(attrs.Operator.Value)
}

func scalarZero(value constant.Value) bool {
	if value == nil {
		return false
	}
	switch value.Kind() {
	case constant.Bool:
		return !constant.BoolVal(value)
	case constant.String:
		return constant.StringVal(value) == ""
	case constant.Unknown:
		return false
	default:
		return constant.Sign(value) == 0
	}
}

func negateValue(value Value) Value {
	if truth, known := value.Bool(); known {
		return Value{Scalar: constant.MakeBool(!truth)}
	}
	if value.Boolean == 0 {
		return Value{}
	}
	value.Negated = !value.Negated
	return value
}

func logicalShortCircuit(operator string, truth, known bool) (bool, bool) {
	if !known {
		return false, false
	}
	if operator == "&&" {
		return false, !truth
	}
	return true, truth
}

// ConditionSource removes the canonical truth wrapper, retaining source
// correspondence without exposing compiler syntax to the exit evaluator.
func (q *Query) ConditionSource(ref pdg.Ref) pdg.Ref {
	if q == nil || ref == 0 || int(ref) > len(q.graph.Nodes) {
		return 0
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	if subject := q.truthOperand(ref, attrs); subject != 0 {
		return subject
	}
	return ref
}
