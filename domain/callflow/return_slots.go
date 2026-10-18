package callflow

func (evaluation *exitEvaluation) assignReturnSlots(point exitPoint, returned string) exitPoint {
	values := evaluation.values[point.values]
	next := values
	next.bindings = cloneValueMap(values.bindings)
	next.cells = cloneValueMap(values.cells)
	for i, output := range evaluation.body.graph.Function.Outputs {
		if output.Symbol == 0 {
			continue
		}
		value := exitValue(returned[i])
		next.bindings[output.Symbol] = value
		if _, tracked := next.cells[output.Symbol]; tracked {
			next.cells[output.Symbol] = joinCellValue(next.cells, output.Symbol, value)
		}
	}
	point.values = evaluation.internValues(next)
	return point
}

func (evaluation *exitEvaluation) completedReturn(value completion) string {
	outputs := evaluation.body.graph.Function.Outputs
	returned := make([]byte, len(outputs))
	bindings := evaluation.values[value.values].bindings
	for i, output := range outputs {
		if output.Symbol != 0 {
			returned[i] = byte(bindings[output.Symbol])
			continue
		}
		returned[i] = byte(evaluation.unnamedReturn(value, i))
	}
	return string(returned)
}

func (evaluation *exitEvaluation) unnamedReturn(value completion, index int) exitValue {
	if len(value.returned) == bodyResultCount(evaluation.body) {
		return exitValue(value.returned[index])
	}
	return abstractValue(evaluation.body.paths.ParameterZero(evaluation.body.graph.Function.Outputs[index]))
}
