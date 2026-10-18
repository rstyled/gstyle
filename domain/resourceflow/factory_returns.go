package resourceflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (a *factoryAnalysis) deferEffect(invocation pdg.Ref, captures []deferflow.Capture, state *factoryState) {
	if a.query.deferredReturns(invocation) {
		return
	}
	a.invalidateExternal(state)
	known := a.query.closeMethod(invocation) || a.query.helper(invocation, Policy{Helpers: a.helpers})
	if !known {
		state.unknownExit = true
	}
	for _, capture := range captures {
		a.invalidate(a.origins(capture.Source, 0), state)
	}
}
func (a *factoryAnalysis) errorRelation(returned, resource pdg.Ref) (bool, bool) {
	if len(a.query.graph.Function.Outputs) < 2 {
		return true, true
	}
	origins := a.origins(returned, 2)
	if !origins.Known || len(origins.Sources) == 0 {
		return false, false
	}
	return a.errorOrigins(resource, origins)
}
func (a *factoryAnalysis) errorOrigins(resource pdg.Ref, origins bindingflow.Origins) (bool, bool) {
	nilOnly, paired := true, true
	for _, source := range origins.Sources {
		if source.Representation == bindingflow.UnchangedRepresentation && a.query.emptySource(source) {
			continue
		}
		nilOnly = false
		if !sameErrorResult(source, resource) {
			paired = false
		}
	}
	return nilOnly, paired
}
func sameErrorResult(source bindingflow.ValueSource, resource pdg.Ref) bool {
	return source.Representation == bindingflow.UnchangedRepresentation && source.Node == resource && source.Result == 2 && !source.Zero
}
