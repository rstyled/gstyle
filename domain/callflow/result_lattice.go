package callflow

// Empty is no finite return evidence. Encoded tuples retain explicit unknown
// slots separately from bottom and can carry arbitrary canonical identities.
func joinReturned(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	a, b := decodeTuple(left), decodeTuple(right)
	if len(a) != len(b) {
		return unknownReturned(max(len(a), len(b)))
	}
	for i := range a {
		a[i] = joinValue(a[i], b[i])
	}
	return encodeTuple(a)
}
func unknownReturned(count int) string { return encodeTuple(make([]exitValue, count)) }

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
