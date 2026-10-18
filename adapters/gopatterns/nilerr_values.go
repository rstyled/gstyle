package gopatterns

import (
	"go/constant"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type nilErrValue = bindingflow.Value

func (e nilErrEvidence) valueRef(ref pdg.Ref) nilErrValue {
	if e.graph == nil {
		return nilErrValue{}
	}
	return e.graph.copies.Evaluate(ref, e.evaluation())
}

func (e nilErrEvidence) booleanRef(ref pdg.Ref) (bool, bool) { return e.valueRef(ref).Bool() }

func (e *nilErrEvidence) assumeRef(ref pdg.Ref, truth bool) {
	if check := e.graph.nilCheck(ref, e.info); check != nil {
		check.isNotNil = check.isNotNil == truth
		e.assumeError(*check)
		return
	}
	value := e.valueRef(ref)
	if value.Boolean != 0 {
		e.truths[value.Boolean] = truth != value.Negated
	}
}

func (e nilErrEvidence) canAssumeRef(ref pdg.Ref) bool {
	return e.valueRef(ref).Boolean != 0 || e.graph.nilCheck(ref, e.info) != nil
}

func (e nilErrEvidence) canSplitRef(ref pdg.Ref) bool { return e.valueRef(ref).Boolean != 0 }

func (e *nilErrEvidence) assumeError(fact nilErrCheck) {
	e.guards = append(append([]nilErrCheck{}, e.guards...), fact)
	if len(fact.path.Fields) == 0 {
		e.values[fact.path.Root] = nilErrValue{Nil: !fact.isNotNil, NonNil: fact.isNotNil}
	}
}

func nilErrZeroType(typ types.Type) nilErrValue {
	if _, generic := types.Unalias(typ).(*types.TypeParam); generic {
		return nilErrValue{}
	}
	switch typ := typ.Underlying().(type) {
	case *types.Basic:
		return nilErrZeroBasic(typ)
	case *types.Array, *types.Struct:
		return nilErrValue{Empty: true}
	default:
		return nilErrValue{Nil: true}
	}
}

func nilErrZeroBasic(typ *types.Basic) nilErrValue {
	if typ.Info()&types.IsBoolean != 0 {
		return nilErrValue{Scalar: constant.MakeBool(false)}
	}
	if typ.Info()&types.IsString != 0 {
		return nilErrValue{Scalar: constant.MakeString("")}
	}
	return nilErrValue{Scalar: constant.MakeInt64(0)}
}

func nilErrAssignableZero(value nilErrValue, typ types.Type) bool {
	if _, generic := types.Unalias(typ).(*types.TypeParam); generic {
		return false
	}
	if _, iface := typ.Underlying().(*types.Interface); iface {
		return value.Nil
	}
	return value.Zero()
}
