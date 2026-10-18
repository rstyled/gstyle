package bindingflow

import "github.com/rstyled/gstyle/domain/pdg"

func (q *Query) singleCellSymbols() (map[pdg.Ref]bool, error) {
	result := map[pdg.Ref]bool{}
	for i, symbol := range q.graph.Symbols {
		single, err := q.booleanSymbolMarker(symbol.Extensions, "gstyle.memory:single-cell")
		if err != nil {
			return nil, err
		}
		if single {
			result[pdg.Ref(i+1)] = true
		}
	}
	return result, nil
}
