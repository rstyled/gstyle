package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// An interface field retains its initializer's dynamic type only while its
// fresh owner stays local and that field is never replaced or addressed.
func (q *assertionGuardQuery) freshFieldType(ref pdg.Ref, assertion *ast.TypeAssertExpr) bool {
	selector, ok := ast.Unparen(assertion.X).(*ast.SelectorExpr)
	if !ok {
		return false
	}
	base, field := q.freshFieldSubject(ref, selector)
	if base == nil {
		return false
	}
	literal := q.freshLiteral(base)
	if literal == nil {
		return false
	}
	value := q.freshLiteralField(literal, field)
	if value == nil {
		return false
	}
	return satisfiesTypeAssertion(concreteInterfaceValueType(q.info.TypeOf(value)), q.info.TypeOf(assertion.Type)) && q.freshFieldUses(base, field)
}

func (q *assertionGuardQuery) freshFieldSubject(ref pdg.Ref, selector *ast.SelectorExpr) (*ast.Ident, *types.Var) {
	base, ok := ast.Unparen(selector.X).(*ast.Ident)
	if !ok {
		return nil, nil
	}
	field := q.directAssertionField(selector)
	if field == nil {
		return nil, nil
	}
	input := q.canonicalFieldInput(ref, selector)
	if input == 0 {
		return nil, nil
	}
	path := q.bindings.Path(input, q.facts)
	root := q.localPath(base, q.refs[selector.X])
	if !q.freshFieldPath(path, root, selector) {
		return nil, nil
	}
	return base, field
}

func (q *assertionGuardQuery) directAssertionField(selector *ast.SelectorExpr) *types.Var {
	selection := q.info.Selections[selector]
	if selection == nil || selection.Kind() != types.FieldVal || len(selection.Index()) != 1 {
		return nil
	}
	field, _ := selection.Obj().(*types.Var)
	return field
}

func (q *assertionGuardQuery) canonicalFieldInput(ref pdg.Ref, selector *ast.SelectorExpr) pdg.Ref {
	input := q.bindings.Operand(ref, 1)
	if input == 0 || q.refs[selector] != input {
		return 0
	}
	if q.bindings.Operand(input, 1) != q.refs[selector.X] {
		return 0
	}
	return input
}

func (q *assertionGuardQuery) freshFieldPath(path, root bindingflow.Path, selector *ast.SelectorExpr) bool {
	if root.Root == 0 || path.Root != root.Root || len(path.Fields) != 1 {
		return false
	}
	return path.Fields[0] == uint32(q.info.Selections[selector].Index()[0]) && q.bindings.StablePath(path, q.facts)
}

func (q *assertionGuardQuery) freshLiteralField(literal *ast.CompositeLit, field *types.Var) ast.Expr {
	for index, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return nil
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok {
			return nil
		}
		if q.info.ObjectOf(key) == field {
			return q.freshFieldInitializer(literal, pair, pdg.Position(index+1))
		}
	}
	return nil
}

func (q *assertionGuardQuery) freshFieldInitializer(literal *ast.CompositeLit, pair *ast.KeyValueExpr, position pdg.Position) ast.Expr {
	owner, entry, value := q.refs[literal], q.refs[pair], q.refs[pair.Value]
	if owner == 0 || entry == 0 || value == 0 {
		return nil
	}
	if q.bindings.Operand(owner, position) != entry || q.bindings.Operand(entry, 2) != value {
		return nil
	}
	return pair.Value
}
