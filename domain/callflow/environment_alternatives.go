package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type environmentAlternatives []environmentProjection

func (alternatives environmentAlternatives) captureWrites() []map[pdg.Ref]exitValue {
	var writes []map[pdg.Ref]exitValue
	for _, alternative := range alternatives {
		writes = append(writes, alternative.writes)
	}
	return writes
}
func (alternatives environmentAlternatives) missesCell(address string) bool {
	for _, alternative := range alternatives {
		if _, found := alternative.heap[address]; !found {
			return true
		}
	}
	return false
}
func (projection *environmentProjection) retainUntouched(alternatives environmentAlternatives, context bindingflow.Evaluation) {
	for address, cell := range projection.heap {
		if projection.fresh[address] || !alternatives.missesCell(address) {
			continue
		}
		cell.Value = joinValue(cell.Value, abstractValue(context.ClosureCells[address]))
		cell.Repeated = cell.Repeated || context.RepeatedCells[address]
		projection.heap[address] = cell
	}
}
