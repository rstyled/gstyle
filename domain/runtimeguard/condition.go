package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

var inverse = map[string]string{"<": ">=", "<=": ">", ">": "<=", ">=": "<", "==": "!=", "!=": "=="}
var reverse = map[string]string{"<": ">", "<=": ">=", ">": "<", ">=": "<=", "==": "==", "!=": "!="}

func (v *graphView) assume(ref pdg.Ref, truth bool, state State) State {
	if value := v.fact(ref, "boolean"); value != "" && (value == "true") != truth {
		return nil
	}
	operator := v.graph.Tables.Text(v.attrs(ref).Operator.Value)
	switch operator {
	case "parentheses":
		return v.assume(v.operand(ref, 0), truth, state)
	case "!":
		return v.assume(v.operand(ref, 0), !truth, state)
	case "&&", "||":
		return v.logical(ref, operator, truth, state)
	default:
		return v.comparison(ref, operator, truth, state)
	}
}

func (v *graphView) logical(ref pdg.Ref, operator string, truth bool, state State) State {
	if (operator == "&&") == truth {
		left := v.assume(v.operand(ref, 0), truth, state)
		if left == nil {
			return nil
		}
		return v.assume(v.operand(ref, 1), truth, left)
	}
	return v.join(v.assume(v.operand(ref, 0), truth, copyState(state)), v.assume(v.operand(ref, 1), truth, copyState(state)))
}

func (v *graphView) comparison(ref pdg.Ref, operator string, truth bool, state State) State {
	if !truth {
		operator = inverse[operator]
	}
	left, right := v.operand(ref, 0), v.operand(ref, 1)
	if symbol := v.symbol(left); symbol != 0 {
		return v.constrain(symbol, operator, v.value(right, state), state)
	}
	if symbol := v.symbol(right); symbol != 0 {
		return v.constrain(symbol, reverse[operator], v.value(left, state), state)
	}
	return state
}

func (v *graphView) constrain(symbol pdg.Ref, operator string, bound Value, state State) State {
	value := v.binding(symbol, state)
	if bound.Nil {
		return v.nilConstraint(symbol, operator, value, state)
	}
	if upperLengthCheck(bound, operator) {
		value.UpperChecked = true
	}
	if exactInteger(bound) {
		value = integerConstraint(value, operator, bound.Lower)
	}
	if emptyRange(value) {
		return nil
	}
	state[symbol] = value
	return state
}

func (v *graphView) nilConstraint(symbol pdg.Ref, operator string, value Value, state State) State {
	if v.kinds[symbol-1] != `"pointer"` {
		return state
	}
	switch operator {
	case "==":
		if value.NonNil {
			return nil
		}
		value.NonNil, value.Nil = false, true
	case "!=":
		if value.Nil {
			return nil
		}
		value.NonNil, value.Nil, value.Guarded = true, false, true
	default:
		return state
	}
	state[symbol] = value
	return state
}

func integerConstraint(value Value, operator string, bound int64) Value {
	switch operator {
	case ">=":
		return lowerConstraint(value, bound)
	case ">":
		return strictLowerConstraint(value, bound)
	case "<":
		return strictUpperConstraint(value, bound)
	case "<=":
		return upperConstraint(value, bound)
	case "==":
		return upperConstraint(lowerConstraint(value, bound), bound)
	default:
		return value
	}
}

func lowerConstraint(value Value, bound int64) Value {
	if !value.HasLower || value.Lower < bound {
		value.Lower = bound
	}
	value.HasLower = true
	return value
}

func upperConstraint(value Value, bound int64) Value {
	if !value.HasUpper || value.Upper > bound {
		value.Upper = bound
	}
	value.HasUpper = true
	return value
}

func upperLengthCheck(bound Value, operator string) bool {
	return bound.Length && (operator == "<" || operator == "<=")
}

func exactInteger(value Value) bool {
	return value.HasLower && value.HasUpper && value.Lower == value.Upper
}

func emptyRange(value Value) bool {
	return value.HasLower && value.HasUpper && value.Lower > value.Upper
}

func strictLowerConstraint(value Value, bound int64) Value {
	if bound == int64(^uint64(0)>>1) {
		return value
	}
	return lowerConstraint(value, bound+1)
}

func strictUpperConstraint(value Value, bound int64) Value {
	if bound == -int64(^uint64(0)>>1)-1 {
		return value
	}
	return upperConstraint(value, bound-1)
}
