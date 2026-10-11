package gopatterns

import (
	"go/types"

	"github.com/shanejonas/cyclo/domain/pdg"
)

// Count distinct recorded variable bindings, not uses, aliases or locations.
// Go's reference-bearing value kinds define the language-specific policy.
func (b *irBuilder) references() {
	policy := b.pool.Policy("cyclo.go-reference-bindings", "1")
	var count uint64
	for object := range b.symbols {
		variable, ok := referenceVariable(object)
		if !ok {
			continue
		}
		typ := types.Unalias(variable.Type())
		if _, generic := typ.(*types.TypeParam); generic {
			b.graph.Function.ReferenceCount = pdg.CountFact{
				Status: pdg.Unknown, Policy: policy,
				Reason: b.pool.Text("Reference classification of a type parameter is unresolved."),
			}
			return
		}
		if referenceType(typ) {
			count++
		}
	}
	b.graph.Function.ReferenceCount = pdg.CountFact{Status: pdg.Known, Policy: policy, Value: count}
}

func referenceType(typ types.Type) bool {
	switch typ.Underlying().(type) {
	case *types.Pointer, *types.Map, *types.Slice, *types.Chan, *types.Signature, *types.Interface:
		return true
	}
	basic, ok := typ.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.UnsafePointer
}

func referenceVariable(object types.Object) (*types.Var, bool) {
	variable, ok := object.(*types.Var)
	return variable, ok && !variable.IsField()
}
