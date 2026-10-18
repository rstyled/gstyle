package callflow

func (evaluation *exitEvaluation) solveCompletions(point exitPoint) Outcomes {
	for {
		evaluation.seen = map[exitPoint]bool{}
		evaluation.changed = false
		result := evaluation.visit(point)
		if !evaluation.changed {
			return evaluation.publishCompletions(result)
		}
	}
}

func (evaluation *exitEvaluation) publishCompletions(result completions) Outcomes {
	var outcomes Outcomes
	for value := range result {
		mode := value.mode
		if mode == Returns && value.recovered {
			mode = RecoversInherited
		}
		outcomes |= mode
		evaluation.publishCapture(value, mode)
		evaluation.publishHeap(value, mode)
		if mode == Returns {
			evaluation.returned = joinReturned(evaluation.returned, evaluation.completedReturn(value))
		}
	}
	return outcomes
}

func (evaluation *exitEvaluation) publishCapture(value completion, mode Outcomes) {
	index := completionIndex(mode)
	if index < 0 {
		return
	}
	values := evaluation.captureTuple(evaluation.completionPoint(value))
	evaluation.captured[index] = joinReturned(evaluation.captured[index], values)
}
