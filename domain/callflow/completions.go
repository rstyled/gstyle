package callflow

// A completion describes one abstract suffix exit before older registrations
// run. The saved result tuple belongs to the return operation, not live cells.
type completion struct {
	values    valueStateID
	returned  string
	mode      Outcomes
	recovered bool
}

type completions map[completion]bool

func (values completions) add(other completions) {
	for value := range other {
		values[value] = true
	}
}

func (values completions) outcomes() Outcomes {
	var result Outcomes
	for value := range values {
		result |= value.mode
	}
	return result
}

func terminalCompletion(mode Outcomes) completions {
	return completions{{mode: mode}: true}
}

func (evaluation *exitEvaluation) finish(point exitPoint, outcomes Outcomes) completions {
	result := terminalCompletion(outcomes & (Aborts | Diverges | UnknownExit))
	delete(result, completion{})
	for _, mode := range pendingExitModes() {
		if outcomes&mode != 0 {
			result[completion{values: point.values, mode: mode, recovered: point.recovered}] = true
		}
	}
	return result
}

func (evaluation *exitEvaluation) completionPoint(value completion) exitPoint {
	return exitPoint{values: value.values, recovered: value.recovered}
}
