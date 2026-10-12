package pdg

import "fmt"

func validateAcyclicFlow(flow *NormalFlow) error {
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
	if consumeFlowBlocks(flow, pending) != count {
		return fmt.Errorf("cyclic flow is outside the scalar normal profile")
	}
	return nil
}

func flowSeeds(flow *NormalFlow, pending []int) []Ref {
	var queue []Ref
	for i, block := range flow.Blocks {
		if block.Reachable && pending[i] == 0 {
			queue = append(queue, Ref(i+1))
		}
	}
	return queue
}

func consumeFlowBlocks(flow *NormalFlow, pending []int) int {
	queue := flowSeeds(flow, pending)
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
