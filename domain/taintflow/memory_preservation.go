package taintflow

import "github.com/rstyled/gstyle/domain/callflow"

func proveMemory(queries map[string]*query) error {
	prepared := make(map[string]callflow.Queries, len(queries))
	for name, query := range queries {
		prepared[name] = query.prepared
	}
	proofs, err := callflow.MemoryPreservationWithQueries(prepared)
	if err != nil {
		return err
	}
	for name, query := range queries {
		query.memoryPreserved = proofs[name]
	}
	return nil
}
