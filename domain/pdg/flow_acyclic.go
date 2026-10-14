package pdg

import "fmt"

func (flow *NormalFlow) validateAcyclicFlow() error {
	pending := make([]int, len(flow.Blocks))
	count := 0
	for _, block := range flow.Blocks {
		if !block.Reachable {
			continue
		}
		count++
		for _, next := range block.Successors {
			pending[next-1]++
		}
	}
	if flow.consumeFlowBlocks(pending) != count {
		return fmt.Errorf("cyclic flow is outside the scalar normal profile")
	}
	return nil
}

func (flow *NormalFlow) flowSeeds(pending []int) []Ref {
	var queue []Ref
	for i, block := range flow.Blocks {
		if block.Reachable && pending[i] == 0 {
			queue = append(queue, Ref(i+1))
		}
	}
	return queue
}

func (flow *NormalFlow) consumeFlowBlocks(pending []int) int {
	queue := flow.flowSeeds(pending)
	for cursor := 0; cursor < len(queue); cursor++ {
		for _, next := range flow.Blocks[queue[cursor]-1].Successors {
			pending[next-1]--
			if pending[next-1] == 0 {
				queue = append(queue, next)
			}
		}
	}
	return len(queue)
}
