package bindingflow

import (
	"go/constant"
	"maps"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func valueQuery(t *testing.T, operators ...string) *Query {
	t.Helper()
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables}
	var attrs []pdg.Attributes
	for i, operator := range operators {
		node := pdg.Node{Category: pdg.Computation, Attributes: pdg.Ref(i + 1)}
		a := pdg.Attributes{Operator: pool.Known(operator)}
		if operator == "read" {
			node.Category = pdg.Read
			a.Reads = []pdg.Access{{Symbol: 1, Position: 1}}
		}
		if operator == "if" {
			node.Category = pdg.Control
			a.Construct = pool.Text("if")
		}
		graph.Nodes = append(graph.Nodes, node)
		attrs = append(attrs, a)
	}
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
	return &Query{representations: map[pdg.Ref]map[pdg.Position]Representation{}, reaching: make([][]pdg.Edge, len(operators)), graph: graph, operands: make([]map[pdg.Position]operand, len(operators)), weak: map[pdg.Ref]bool{}, fields: map[pdg.Ref][]uint32{}, required: map[pdg.Ref][]pdg.Position{}, allocations: map[pdg.Ref]pdg.Ref{}}
}
func valueEdge(q *Query, source, target pdg.Ref, position pdg.Position) {
	// Indexing needs an interned semantic kind; extend only this test fixture.
	q.graph.Tables.Strings = append(q.graph.Tables.Strings, "source-value")
	q.index(pdg.Edge{Source: source, Target: target, Kind: pdg.Data, Subkind: pdg.Text(len(q.graph.Tables.Strings)), Position: position})
}
func boolValue(code int) Value {
	if code == 0 {
		return Value{}
	}
	return Value{Scalar: constant.MakeBool(code == 2)}
}
func TestLogicalValuesAgainstThreeStateTruthTable(t *testing.T) {
	for _, operator := range []string{"&&", "||"} {
		q := valueQuery(t, "leaf", "leaf", operator)
		valueEdge(q, 1, 3, 1)
		valueEdge(q, 2, 3, 2)
		for left := 0; left < 3; left++ {
			for right := 0; right < 3; right++ {
				context := Evaluation{Facts: knownValues(map[pdg.Ref]Value{1: boolValue(left), 2: boolValue(right)})}
				truth, known := q.Evaluate(3, context).Bool()
				wantKnown := left != 0 && right != 0
				want := left == 2 && right == 2
				if operator == "&&" && (left == 1 || right == 1) {
					wantKnown = true
					want = false
				}
				if operator == "||" {
					want = left == 2 || right == 2
					if want {
						wantKnown = true
					}
				}
				if known != wantKnown || known && truth != want {
					t.Fatalf("%d %s %d: (%v,%v),want (%v,%v)", left, operator, right, truth, known, want, wantKnown)
				}
			}
		}
	}
}
func TestComparisonValuesAndInvalidKinds(t *testing.T) {
	for _, tc := range []struct {
		name, operator string
		left, right    constant.Value
		truth, known   bool
	}{
		{"equal integers", "==", constant.MakeInt64(2), constant.MakeInt64(2), true, true},
		{"ordered integers", "<", constant.MakeInt64(1), constant.MakeInt64(2), true, true},
		{"mixed numbers", "<=", constant.MakeInt64(2), constant.MakeFloat64(2), true, true},
		{"strings", "!=", constant.MakeString("a"), constant.MakeString("b"), true, true},
		{"boolean equality", "==", constant.MakeBool(true), constant.MakeBool(false), false, true},
		{"invalid boolean order", "<", constant.MakeBool(true), constant.MakeBool(false), false, false},
		{"incompatible kinds", "==", constant.MakeBool(false), constant.MakeInt64(0), false, false},
		{"missing scalar", "==", nil, constant.MakeInt64(0), false, false},
		{"unsupported arithmetic", "+", constant.MakeInt64(1), constant.MakeInt64(2), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := valueQuery(t, "leaf", "leaf", tc.operator)
			valueEdge(q, 1, 3, 1)
			valueEdge(q, 2, 3, 2)
			got, known := q.Evaluate(3, Evaluation{Facts: knownValues(map[pdg.Ref]Value{1: {Scalar: tc.left}, 2: {Scalar: tc.right}})}).Bool()
			if known != tc.known || known && got != tc.truth {
				t.Fatalf("got (%v,%v),want (%v,%v)", got, known, tc.truth, tc.known)
			}
		})
	}
}
func TestNilPredicateUsesCanonicalWrappers(t *testing.T) {
	q := valueQuery(t, "read", "leaf", "==", "!", "parentheses", "if")
	valueEdge(q, 1, 3, 1)
	valueEdge(q, 2, 3, 2)
	valueEdge(q, 3, 4, 1)
	valueEdge(q, 4, 5, 1)
	valueEdge(q, 5, 6, 1)
	facts := knownValues(map[pdg.Ref]Value{2: {Nil: true}})
	if subject, notNil := q.NilPredicate(6, facts); subject != 1 || !notNil {
		t.Fatalf("subject=%d notNil=%v", subject, notNil)
	}
	context := Evaluation{Facts: facts, Bindings: map[pdg.Ref]Value{1: {NonNil: true}}}
	if truth, known := q.Evaluate(6, context).Bool(); !known || !truth {
		t.Fatalf("predicate=(%v,%v)", truth, known)
	}
	delete(q.operands[3], 1)
	if subject, _ := q.NilPredicate(6, facts); subject != 0 {
		t.Fatal("missing negation input falls back to another path")
	}
	if _, known := q.Evaluate(6, context).Bool(); known {
		t.Fatal("missing negation input invents truth")
	}
}
func TestSymbolicBooleansAndStateRemainUnchanged(t *testing.T) {
	q := valueQuery(t, "read", "!")
	valueEdge(q, 1, 2, 1)
	context := Evaluation{Bindings: map[pdg.Ref]Value{}, Truths: map[pdg.Ref]bool{}, Facts: Facts{BooleanSymbols: map[pdg.Ref]bool{1: true}}}
	got := q.Evaluate(2, context)
	if got.Boolean != 1 || !got.Negated {
		t.Fatalf("symbolic negation=%+v", got)
	}
	context.Bindings[1] = Value{Boolean: 2, Negated: true}
	context.Truths[2] = true
	before := maps.Clone(context.Bindings)
	if truth, known := q.Evaluate(2, context).Bool(); !known || !truth {
		t.Fatalf("copied predicate=(%v,%v)", truth, known)
	}
	if !reflect.DeepEqual(before, context.Bindings) {
		t.Fatal("evaluation modifies input state")
	}
}
func TestCyclicOrMissingValueOperandsStayUnknown(t *testing.T) {
	q := valueQuery(t, "parentheses")
	valueEdge(q, 1, 1, 1)
	if _, known := q.Evaluate(1, Evaluation{}).Bool(); known {
		t.Fatal("cycle invents a value")
	}
	if subject, _ := q.NilPredicate(1, Facts{}); subject != 0 {
		t.Fatal("cyclic wrapper invents a predicate")
	}
	for _, ref := range []pdg.Ref{0, 2} {
		if got := q.Evaluate(ref, Evaluation{}); got.Scalar != nil || got.Nil || got.Boolean != 0 {
			t.Fatalf("missing ref %d: %+v", ref, got)
		}
	}
}
func TestConversionsRequireActualOperands(t *testing.T) {
	q := valueQuery(t, "leaf", "conversion")
	valueEdge(q, 1, 2, 1)
	facts := knownValues(map[pdg.Ref]Value{1: {Nil: true}})
	facts.Nodes[2] = Fact{Conversion: NilConversion}
	context := Evaluation{Facts: facts}
	if !q.Evaluate(2, context).Nil {
		t.Fatal("nil conversion loses compiler nil input")
	}
	delete(q.operands[1], 1)
	if q.Evaluate(2, context).Nil {
		t.Fatal("conversion uses source syntax after operand removal")
	}
}

func TestUnknownOperatorDoesNotActLikeItsReason(t *testing.T) {
	q := valueQuery(t, "leaf", "!")
	valueEdge(q, 1, 2, 1)
	attrs := q.graph.NodeAttributes(q.graph.Nodes[1])
	attrs.Operator.Status = pdg.Unknown
	input := []pdg.Attributes{q.graph.NodeAttributes(q.graph.Nodes[0]), attrs}
	q.graph.Attributes = nil
	q.graph.AttributeTables = pdg.AttributeTables{}
	for i := range q.graph.Nodes {
		q.graph.Nodes[i].Attributes = pdg.Ref(i + 1)
	}
	if err := pdg.PackAttributes(q.graph, input); err != nil {
		t.Fatal(err)
	}
	context := Evaluation{Facts: knownValues(map[pdg.Ref]Value{1: {Scalar: constant.MakeBool(false)}})}
	if _, known := q.Evaluate(2, context).Bool(); known {
		t.Fatal("unknown operator reason supplies semantics")
	}
}
func TestExactValueZerosRespectRepresentation(t *testing.T) {
	for _, tc := range []struct {
		value Value
		zero  bool
	}{
		{Value{}, false}, {Value{Nil: true}, true}, {Value{NonNil: true}, false}, {Value{Empty: true}, true},
		{Value{Scalar: constant.MakeBool(false)}, true}, {Value{Scalar: constant.MakeString("")}, true},
		{Value{Scalar: constant.MakeInt64(1)}, false}, {Value{Scalar: constant.MakeUnknown()}, false},
	} {
		if got := tc.value.Zero(); got != tc.zero {
			t.Fatalf("%+v zero=%v want %v", tc.value, got, tc.zero)
		}
	}
}

func knownValues(values map[pdg.Ref]Value) Facts {
	facts := Facts{Nodes: map[pdg.Ref]Fact{}}
	for ref, value := range values {
		facts.Nodes[ref] = Fact{Known: true, Value: value}
	}
	return facts
}

func TestAbsentBooleanIdentityCannotUseZeroTruthSlot(t *testing.T) {
	q := valueQuery(t, "read")
	context := Evaluation{Bindings: map[pdg.Ref]Value{1: {}}, Truths: map[pdg.Ref]bool{0: true}}
	if _, known := q.Evaluate(1, context).Bool(); known {
		t.Fatal("absent boolean identity uses zero slot")
	}
}
