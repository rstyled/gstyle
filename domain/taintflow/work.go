package taintflow

import "errors"

const summaryWorkLimit = 100_000_000

var ErrWorkLimit = errors.New("taint summary work limit exceeded")

type analysisWork struct {
	remaining int
	exhausted bool
}

func (work *analysisWork) spend(amount int) bool {
	if work == nil {
		return true
	}
	if work.exhausted || amount > work.remaining {
		work.exhausted = true
		return false
	}
	work.remaining -= amount
	return true
}

func (work *analysisWork) available() bool {
	return work == nil || !work.exhausted
}

func (store state) joinValues(left, right value) value {
	if !store.work.spend(valueSize(left) + valueSize(right) + 1) {
		return value{}
	}
	return store.unions.join(left, right)
}

func valueSize(item value) int {
	return len(item.Data) + len(item.Objects) + len(item.Functions)
}
