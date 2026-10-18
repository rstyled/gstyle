package callflow

func (evaluation *exitEvaluation) assignReturnSlots(point exitPoint, returned string) exitPoint {
	slots := decodeTuple(returned)
	values := evaluation.values[point.values]
	next := values
	next.bindings = cloneValueMap(values.liveBindings())
	next.cells = cloneValueMap(values.cells)
	for i, output := range evaluation.body.graph.Function.Outputs {
		if output.Symbol == 0 {
			continue
		}
		value := slots[i]
		next.bindings[output.Symbol] = value
		if _, tracked := next.cells[output.Symbol]; tracked {
			next.cells[output.Symbol] = joinCellValue(next.cells, output.Symbol, value)
		}
	}
	next.heap = cloneEnvironmentHeap(next.heap)
	for i, output := range evaluation.body.graph.Function.Outputs {
		if address := next.locations[output.Symbol]; address != "" {
			writeEnvironmentCell(next.heap, address, slots[i])
		}
	}
	point.values = evaluation.internValues(next)
	return point
}

func (evaluation *exitEvaluation) completedReturn(value completion) string {
	outputs := evaluation.body.graph.Function.Outputs
	returned := make([]exitValue, len(outputs))
	bindings := evaluation.values[value.values].liveBindings()
	for i, output := range outputs {
		if output.Symbol != 0 {
			returned[i] = bindings[output.Symbol]
			continue
		}
		returned[i] = evaluation.unnamedReturn(value, i)
	}
	return encodeTuple(returned)
}

func (evaluation *exitEvaluation) unnamedReturn(value completion, index int) exitValue {
	slots := decodeTuple(value.returned)
	if len(slots) == bodyResultCount(evaluation.body) {
		return slots[index]
	}
	return abstractValue(evaluation.body.paths.ParameterZero(evaluation.body.graph.Function.Outputs[index]))
}
