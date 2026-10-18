package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/constant"
	"go/token"
	"testing"
)

func TestSymbolicBooleanComparisonTruthTable(t *testing.T) {
	for _, operator := range []string{"==", "!="} {
		for _, negated := range []bool{false, true} {
			for _, reverse := range []bool{false, true} {
				for _, literal := range []bool{false, true} {
					query := valueQuery(t, "read", "!", "leaf", operator)
					valueEdge(query, 1, 2, 1)
					subject := pdg.Ref(1)
					if negated {
						subject = 2
					}
					left, right := subject, pdg.Ref(3)
					if reverse {
						left, right = right, left
					}
					valueEdge(query, left, 4, 1)
					valueEdge(query, right, 4, 2)
					context := Evaluation{Facts: knownValues(map[pdg.Ref]Value{3: {Scalar: constant.MakeBool(literal)}})}
					context.Facts.BooleanSymbols = map[pdg.Ref]bool{1: true}
					result := query.Evaluate(4, context)
					for _, input := range []bool{false, true} {
						truth, known := result.resolveTruth(map[pdg.Ref]bool{1: input}).Bool()
						operand := input != negated
						want := operand == literal
						if operator == "!=" {
							want = !want
						}
						if !known || truth != want {
							t.Errorf("%s negated=%v reverse=%v literal=%v input=%v: %v/%v want %v", operator, negated, reverse, literal, input, truth, known, want)
						}
					}
				}
			}
		}
	}
}

func TestBooleanComparisonDoesNotGuessUnknownRelations(t *testing.T) {
	symbol := Value{Boolean: 1}
	for _, tc := range []struct {
		operator    token.Token
		left, right Value
	}{
		{token.LSS, symbol, Value{Scalar: constant.MakeBool(true)}},
		{token.EQL, symbol, Value{Boolean: 2}},
		{token.NEQ, symbol, Value{}},
		{token.EQL, symbol, Value{Scalar: constant.MakeInt64(1)}},
	} {
		got := booleanComparison(tc.operator, tc.left, tc.right)
		if got.Boolean != 0 || got.Scalar != nil || got.Nil || got.NonNil {
			t.Errorf("unknown relation acquires fact: %+v", got)
		}
	}
}
