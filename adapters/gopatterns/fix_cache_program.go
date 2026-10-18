package gopatterns

import "github.com/rstyled/gstyle/domain/callflow"

// A revision reuses only unaffected graph inventories. Changed dependencies
// and reverse importers are registered in dependency order after rebuilding.
func (e *FixExtractor) programQueries(affected map[string]bool) (*callflow.Program, error) {
	program := callflow.NewProgram()
	if len(affected) == 0 {
		return program, nil
	}
	for name, record := range e.records {
		if affected[name] {
			continue
		}
		if err := e.registerProgramRecord(program, record); err != nil {
			return nil, err
		}
	}
	return program, nil
}

func (e *FixExtractor) registerProgramRecord(program *callflow.Program, record sourcePackageRecord) error {
	for _, source := range record.facts.Callables {
		graph := *source
		graph.Tables = e.pool.Tables
		if err := program.Add(&graph); err != nil {
			return err
		}
	}
	return nil
}
