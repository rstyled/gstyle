package application

import "encoding/json"

type ControlSearchResult struct {
	SearchResult
	Revision   uint64 `json:"revision"`
	Refreshing bool   `json:"refreshing"`
	Error      string `json:"error,omitempty"`
}

func parseSearch(params json.RawMessage) (controlCommand, *controlError) {
	query := SearchQuery{Limit: 20}
	if err := decodeNamedParams(params, &query); err != nil {
		return controlCommand{}, err
	}
	if err := query.Validate(); err != nil {
		return controlCommand{}, invalidParams(err.Error())
	}
	return controlCommand{action: searchAction, search: query}, nil
}

func (m Model) search(query SearchQuery) (ControlSearchResult, *controlError) {
	if err := query.Validate(); err != nil {
		return ControlSearchResult{}, invalidParams(err.Error())
	}
	result := searchRankedReport(m.report, query)
	response := ControlSearchResult{SearchResult: result, Revision: m.revision, Refreshing: m.refreshing}
	if m.err != nil {
		response.Error = m.err.Error()
	}
	return response, nil
}
