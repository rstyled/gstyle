package callflow

// Ordinary observations depend on forward execution states. They do not consume
// suffix completions or deferred invocations after the current frame exits.
// Keep the same value transfers and visit each reachable state once.
func (evaluation *exitEvaluation) observe(point exitPoint) {
	if evaluation.body.canceled() || evaluation.seen[point] {
		return
	}
	evaluation.seen[point] = true
	evaluation.step(point)
}
