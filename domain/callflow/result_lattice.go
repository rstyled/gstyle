package callflow

import "strings"

// Empty is no finite return evidence. A nonempty tuple owns one nil/Boolean
// shape per result; zero bytes are explicit unknown shapes, not bottom.
func joinReturned(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if len(left) != len(right) {
		return unknownReturned(max(len(left), len(right)))
	}
	result := []byte(left)
	for i := range result {
		if result[i] != right[i] {
			result[i] = byte(unknownValue)
		}
	}
	return string(result)
}

func unknownReturned(count int) string { return strings.Repeat("\x00", count) }

func (pass exitPass) returned(old, next string) string {
	if pass != finitePass {
		return old
	}
	return joinReturned(old, next)
}

func bodyResultCount(body *exitBody) int {
	if body == nil {
		return 0
	}
	return len(body.graph.Function.Outputs)
}
