package callflow

// Count both visited program points and the facts retained by their states.
// Independent branches can otherwise retain exponentially many snapshots.
const ordinaryObservationWorkLimit = 1 << 16

// Ordinary observations depend on forward execution states. They do not consume
// suffix completions or deferred invocations after the current frame exits.
// Keep the same value transfers and visit each reachable state once.
func (evaluation *exitEvaluation) observe(point exitPoint) {
	if evaluation.observationLimit || evaluation.body.canceled() || evaluation.seen[point] {
		return
	}
	if !evaluation.reserveObservation(point) {
		return
	}
	evaluation.seen[point] = true
	evaluation.step(point)
}

func (evaluation *exitEvaluation) reserveObservation(point exitPoint) bool {
	values := evaluation.values[point.values]
	work := 1 + len(values.heap) + len(values.locations) + len(values.bindings) + len(values.nodes) + len(values.tuples) + len(values.cells) + len(values.unorderedReads)
	if work > ordinaryObservationWorkLimit-evaluation.observationWork {
		evaluation.observationLimit = true
		return false
	}
	evaluation.observationWork += work
	return true
}
