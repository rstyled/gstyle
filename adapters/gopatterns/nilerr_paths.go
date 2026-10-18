package gopatterns

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/types"
	"maps"
)

const nilErrPathBudget = 256

type nilErrEvidence struct {
	checked   bindingflow.Path
	positions []int
	info      *types.Info
	graph     *nilErrGraph
	values    map[pdg.Ref]nilErrValue
	truths    map[pdg.Ref]bool
	results   []types.Object
	guards    []nilErrCheck
}

func (e nilErrEvidence) clone() nilErrEvidence {
	copy := e
	copy.values = maps.Clone(e.values)
	copy.truths = maps.Clone(e.truths)
	return copy
}
