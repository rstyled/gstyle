package callflow

import "github.com/rstyled/gstyle/domain/pdg"

// Unknown external stores use the same canonical may-write inventory as normal
// calls. Missing query evidence cannot retain exact live bindings.
func (evaluation *exitEvaluation) rawValueTransfer(point exitPoint, ref pdg.Ref) exitPoint {
	if evaluation.body.paths == nil {
		values := evaluation.values[point.values]
		values.bindings = nil
		values.cells = unknownCellValues(values.cells)
		point.values = evaluation.internValues(values)
		return point
	}
	values := evaluation.values[point.values]
	writes := evaluation.body.paths.WriteValues(ref, evaluation.valueContext(point))
	next := values
	next.bindings = nextValueBindings(values.bindings, writes)
	next.cells = evaluation.nextCellValues(values.cells, writes)
	point.values = evaluation.internValues(next)
	return point
}
