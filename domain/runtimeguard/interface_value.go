package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

func nilableKind(kind string) bool {
	return kind == `"pointer"` || kind == `"interface"`
}

// An interface holding a typed nil pointer is non-nil. Keep its whole-value
// nil state separate from pointer identity and payload nilness.
func (v *graphView) interfaceAssignment(ref, symbol pdg.Ref, value Value) Value {
	if v.kinds[symbol-1] != `"interface"` {
		return value
	}
	if v.fact(ref, "zero-initialized") == "true" {
		return Value{Nil: true}
	}
	attrs := v.attrs(ref)
	if !singleValueWrite(v.graph.Nodes[ref-1].Category, v.graph.Tables.Text(attrs.WriteForm), len(attrs.Writes)) {
		return Value{}
	}
	source := v.operand(ref, 0)
	if v.fact(source, "nil") == "true" {
		return Value{Nil: true}
	}
	return interfaceSourceValue(v.fact(source, "interface-value"), value)
}

func interfaceSourceValue(kind string, value Value) Value {
	switch kind {
	case `"concrete"`, `"boxed"`:
		return Value{NonNil: true}
	case `"interface"`:
		return Value{Nil: value.Nil, NonNil: value.NonNil}
	default:
		return Value{}
	}
}
