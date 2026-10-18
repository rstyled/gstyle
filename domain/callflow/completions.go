package callflow

import "iter"

// A completion describes one abstract suffix exit before older registrations
// run. The saved result tuple belongs to the return operation, not live cells.
type completion struct {
	values    valueStateID
	returned  string
	mode      Outcomes
	recovered bool
}

// Most control-flow points have one completion. Keep it inline; allocate a
// set only when paths produce distinct completions.
type completions struct {
	single  completion
	many    map[completion]bool
	present bool
}

func singleCompletion(value completion) completions {
	return completions{single: value, present: true}
}

func (values *completions) insert(value completion) bool {
	if values.many != nil {
		found := values.many[value]
		values.many[value] = true
		return !found
	}
	if !values.present {
		*values = singleCompletion(value)
		return true
	}
	if values.single == value {
		return false
	}
	values.many = map[completion]bool{values.single: true, value: true}
	return true
}

func (values completions) all() iter.Seq[completion] {
	return func(yield func(completion) bool) {
		if values.many != nil {
			for value := range values.many {
				if !yield(value) {
					return
				}
			}
			return
		}
		if values.present {
			yield(values.single)
		}
	}
}

func (values *completions) add(other completions) bool {
	changed := false
	for value := range other.all() {
		changed = values.insert(value) || changed
	}
	return changed
}

func (values *completions) removeZero() {
	if values.many != nil {
		delete(values.many, completion{})
		return
	}
	if values.single == (completion{}) {
		values.present = false
	}
}

func (values completions) outcomes() Outcomes {
	var result Outcomes
	for value := range values.all() {
		result |= value.mode
	}
	return result
}

func terminalCompletion(mode Outcomes) completions {
	return singleCompletion(completion{mode: mode})
}

func (evaluation *exitEvaluation) finish(point exitPoint, outcomes Outcomes) completions {
	result := terminalCompletion(outcomes & (Aborts | Diverges | UnknownExit))
	result.removeZero()
	for _, mode := range pendingExitModes() {
		if outcomes&mode != 0 {
			result.insert(completion{values: point.values, mode: mode, recovered: point.recovered})
		}
	}
	return result
}

func (evaluation *exitEvaluation) completionPoint(value completion) exitPoint {
	return exitPoint{values: value.values, recovered: value.recovered}
}
