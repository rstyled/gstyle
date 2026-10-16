package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

// Exact evaluated operands can reject an edge even when neither operand is a
// binding. In particular, len of a known empty collection equals zero.
func (v *graphView) successorState(condition pdg.Ref, truth bool, state State) State {
	if condition == 0 {
		return state
	}
	if value, known := v.knownBoolean(condition, state); known && value != truth {
		return nil
	}
	return v.assume(condition, truth, state)
}

func (v *graphView) knownBoolean(ref pdg.Ref, state State) (bool, bool) {
	if value := v.fact(ref, "boolean"); value == "true" || value == "false" {
		return value == "true", true
	}
	operator := operatorText(v.graph.Tables, v.attrs(ref).Operator)
	switch operator {
	case "parentheses":
		return v.knownBoolean(v.operand(ref, 0), state)
	case "!":
		value, known := v.knownBoolean(v.operand(ref, 0), state)
		return !value, known
	case "&&", "||":
		return v.knownLogical(ref, operator, state)
	default:
		return v.knownComparison(ref, operator, state)
	}
}

func (v *graphView) knownLogical(ref pdg.Ref, operator string, state State) (bool, bool) {
	left, lk := v.knownBoolean(v.operand(ref, 0), state)
	right, rk := v.knownBoolean(v.operand(ref, 1), state)
	if lk && rk {
		return logicalBoolean(operator, left, right), true
	}
	if lk {
		return shortCircuitBoolean(operator, left)
	}
	if rk {
		return shortCircuitBoolean(operator, right)
	}
	return false, false
}

func logicalBoolean(operator string, left, right bool) bool {
	if operator == "&&" {
		return left && right
	}
	return left || right
}
func shortCircuitBoolean(operator string, value bool) (bool, bool) {
	if operator == "&&" {
		return false, !value
	}
	return true, value
}

func (v *graphView) knownComparison(ref pdg.Ref, operator string, state State) (bool, bool) {
	left, right := v.value(v.operand(ref, 0), state), v.value(v.operand(ref, 1), state)
	if !exactInteger(left) || !exactInteger(right) {
		return false, false
	}
	return compareExactIntegers(operator, left.Lower, right.Lower)
}
func compareExactIntegers(operator string, left, right int64) (bool, bool) {
	if operator == "==" {
		return left == right, true
	}
	if operator == "!=" {
		return left != right, true
	}
	return compareExactOrder(operator, left, right)
}
func compareExactOrder(operator string, left, right int64) (bool, bool) {
	switch operator {
	case "<":
		return left < right, true
	case "<=":
		return left <= right, true
	case ">":
		return left > right, true
	case ">=":
		return left >= right, true
	default:
		return false, false
	}
}
