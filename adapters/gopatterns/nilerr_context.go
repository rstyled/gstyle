package gopatterns

import "go/types"

func (e *nilErrEvidence) initializeResults(signature *types.Signature) {
	e.results = signatureResults(signature)
	for i, result := range e.results {
		if result.Name() == "" {
			continue
		}
		e.values[e.resultSymbol(i)] = nilErrZeroType(result.Type())
	}
}
func signatureResults(signature *types.Signature) []types.Object {
	results := make([]types.Object, signature.Results().Len())
	for i := range results {
		results[i] = signature.Results().At(i)
	}
	return results
}
